package web

import (
	"bufio"
	"errors"
	"net"
	"server/log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Adapted from LT 8a2173280ac75070e77708280273e2025282eb57.
const sniffTimeout = 10 * time.Second
const tlsRecordHandshake = 0x16
const tlsErrorQuietPeriod = 10 * time.Minute

var serverErrors = &serverErrorLog{out: log.TLogln}

// serverErrorLog forwards net/http server errors to the TorrServer log. TLS handshake
// failures are logged once per client per quiet period: a browser that rejects a
// self-signed cert retries every second.
type serverErrorLog struct {
	out func(v ...any)

	mu   sync.Mutex
	seen map[string]time.Time
}

func (l *serverErrorLog) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if host, reason, ok := parseTLSHandshakeError(msg); ok {
		if !l.firstInPeriod(host) {
			return len(p), nil
		}
		msg = "TLS handshake failed from " + host + ": " + reason
		if strings.Contains(reason, "certificate") {
			msg += " (the client does not trust the certificate; with a self-signed cert accept it " +
				"in the browser, or use --sslcert/--sslkey with a trusted one)"
		}
		msg += "; further handshake errors from this client are hidden for " + tlsErrorQuietPeriod.String()
	}
	l.out(msg)
	return len(p), nil
}

func (l *serverErrorLog) firstInPeriod(host string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if last, ok := l.seen[host]; ok && now.Sub(last) < tlsErrorQuietPeriod {
		return false
	}
	if l.seen == nil || len(l.seen) >= 1024 { // bound memory against scanners
		l.seen = make(map[string]time.Time)
	}
	l.seen[host] = now
	return true
}

// parseTLSHandshakeError parses net/http's "http: TLS handshake error from ADDR: REASON".
func parseTLSHandshakeError(msg string) (host, reason string, ok bool) {
	rest, ok := strings.CutPrefix(msg, "http: TLS handshake error from ")
	if !ok {
		return "", "", false
	}
	addr, reason, ok := strings.Cut(rest, ": ")
	if !ok {
		return "", "", false
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	return host, reason, true
}

// splitTLS splits connections accepted on inner by their first byte: TLS handshakes go
// to tlsLn, anything else (plain HTTP sent to the HTTPS port) goes to plainLn.
// inner is closed once both returned listeners are closed.
func splitTLS(inner net.Listener) (tlsLn, plainLn net.Listener) {
	var open atomic.Int32
	open.Store(2)
	var pendingMu sync.Mutex
	pending := make(map[net.Conn]bool)
	closed := false
	onClose := func() {
		if open.Add(-1) == 0 {
			inner.Close()
			pendingMu.Lock()
			closed = true
			for conn := range pending {
				_ = conn.Close()
			}
			pendingMu.Unlock()
		}
	}
	t := newConnQueue(inner.Addr(), onClose)
	p := newConnQueue(inner.Addr(), onClose)
	slots := make(chan struct{}, 256)
	go func() {
		for {
			c, err := inner.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				log.TLogln("https accept error:", err)
				time.Sleep(50 * time.Millisecond) // e.g. out of file descriptors
				continue
			}
			// sniff in a goroutine so a silent client can't stall the accept loop
			select {
			case slots <- struct{}{}:
				pendingMu.Lock()
				if closed {
					pendingMu.Unlock()
					<-slots
					_ = c.Close()
					return
				}
				pending[c] = true
				pendingMu.Unlock()
				go func() {
					defer func() { pendingMu.Lock(); delete(pending, c); pendingMu.Unlock(); <-slots }()
					routeConn(c, t, p)
				}()
			default:
				c.Close()
			}
		}
	}()
	return t, p
}

func routeConn(c net.Conn, tlsQ, plainQ *connQueue) {
	br := bufio.NewReader(c)
	c.SetReadDeadline(time.Now().Add(sniffTimeout))
	first, err := br.Peek(1)
	c.SetReadDeadline(time.Time{})
	if err != nil {
		c.Close()
		return
	}
	pc := &peekedConn{Conn: c, r: br}
	if first[0] == tlsRecordHandshake {
		tlsQ.push(pc)
	} else {
		plainQ.push(pc)
	}
}

// peekedConn replays bytes already buffered while sniffing.
type peekedConn struct {
	net.Conn
	r *bufio.Reader
}

func (c *peekedConn) Read(p []byte) (int, error) { return c.r.Read(p) }

// connQueue is a net.Listener fed by splitTLS.
type connQueue struct {
	conns   chan net.Conn
	done    chan struct{}
	once    sync.Once
	addr    net.Addr
	onClose func()
}

func newConnQueue(addr net.Addr, onClose func()) *connQueue {
	return &connQueue{conns: make(chan net.Conn), done: make(chan struct{}), addr: addr, onClose: onClose}
}

func (q *connQueue) push(c net.Conn) {
	select {
	case q.conns <- c:
	case <-q.done:
		c.Close()
	}
}

func (q *connQueue) Accept() (net.Conn, error) {
	select {
	case c := <-q.conns:
		return c, nil
	case <-q.done:
		return nil, net.ErrClosed
	}
}

func (q *connQueue) Close() error {
	q.once.Do(func() {
		close(q.done)
		q.onClose()
	})
	return nil
}

func (q *connQueue) Addr() net.Addr { return q.addr }
