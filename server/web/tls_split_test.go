package web

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestTLSSplitReplaysBytesAndStopsSilentClients(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	tlsLn, plainLn := splitTLS(ln)
	t.Cleanup(func() { tlsLn.Close(); plainLn.Close() })
	for _, tc := range []struct {
		listener net.Listener
		data     []byte
	}{{tlsLn, []byte{0x16, 0x03, 0x03}}, {plainLn, []byte("GET / HTTP/1.1\r\n")}} {
		client, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
		if err != nil {
			t.Fatal(err)
		}
		client.SetDeadline(time.Now().Add(time.Second))
		if _, err := client.Write(tc.data); err != nil {
			t.Fatal(err)
		}
		accepted := make(chan net.Conn, 1)
		go func() { c, _ := tc.listener.Accept(); accepted <- c }()
		select {
		case c := <-accepted:
			if c == nil {
				t.Fatal("closed before routing")
			}
			c.SetReadDeadline(time.Now().Add(time.Second))
			b := make([]byte, len(tc.data))
			_, err := io.ReadFull(c, b)
			c.Close()
			client.Close()
			if err != nil || string(b) != string(tc.data) {
				t.Fatal("sniff lost bytes", err, b)
			}
		case <-time.After(time.Second):
			client.Close()
			t.Fatal("routing stalled")
		}
	}
	silent, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer silent.Close()
	tlsLn.Close()
	plainLn.Close()
	silent.SetReadDeadline(time.Now().Add(time.Second))
	var b [1]byte
	_, err = silent.Read(b[:])
	if err == nil {
		t.Fatal("silent connection remained open")
	}
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		t.Fatal("shutdown did not close pending sniff")
	}
}
