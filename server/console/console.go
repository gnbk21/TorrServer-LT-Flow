// Package console renders append-only console output. It never touches torrent
// state, reads stdin, or emits cursor movement / screen clearing commands.
package console

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

func Validate(mode string, interval int) error {
	switch mode {
	case "auto", "plain", "off":
	default:
		return fmt.Errorf("--console must be auto, plain, or off")
	}
	if interval != 0 && (interval < 5 || interval > 3600) {
		return fmt.Errorf("--console-interval must be 0 (disabled) or 5..3600 seconds")
	}
	return nil
}

var escapes = regexp.MustCompile("\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\x07\x1b]*(?:\x07|\x1b\\\\))")

// Clean prevents metadata and log messages from issuing terminal commands.
func Clean(s string) string {
	s = escapes.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r == '\t' {
			return ' '
		}
		if r == '\n' {
			return r
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return '?'
		}
		return r
	}, s)
}

type Writer struct {
	mu    sync.Mutex
	out   io.Writer
	color bool
	now   func() time.Time
}

func New(out io.Writer, color bool) *Writer {
	return &Writer{out: out, color: color, now: time.Now}
}

func (w *Writer) paint(code, s string) string {
	if !w.color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func level(message string) string {
	lower := strings.ToLower(message)
	for _, prefix := range []string{"error", "cannot ", "failed ", "flow startup error", "flow service error", "recovered from panic"} {
		if strings.HasPrefix(lower, prefix) {
			return "ERROR"
		}
	}
	for _, prefix := range []string{"warning", "invalid ", "port "} {
		if strings.HasPrefix(lower, prefix) {
			return "WARN"
		}
	}
	return "INFO"
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	message := strings.TrimSuffix(Clean(string(p)), "\n")
	lvl := level(message)
	for _, candidate := range []string{"INFO", "WARN", "ERROR"} {
		if rest, ok := strings.CutPrefix(message, "["+candidate+"] "); ok {
			lvl, message = candidate, rest
			break
		}
	}
	code := map[string]string{"INFO": "36", "WARN": "33", "ERROR": "31"}[lvl]
	prefix := w.paint("90", w.now().Format("15:04:05")) + " " + w.paint(code, fmt.Sprintf("%-5s", lvl)) + " "
	var b strings.Builder
	for _, line := range strings.Split(message, "\n") {
		b.WriteString(prefix + line + "\n")
	}
	n, err := io.WriteString(w.out, b.String())
	if err != nil {
		return 0, err
	}
	if n != b.Len() {
		return 0, io.ErrShortWrite
	}
	return len(p), nil
}

type Section struct {
	Title string
	Rows  []string
}

// Panel is a single write, so concurrent log messages cannot split its sections.
func (w *Writer) Panel(title string, sections []Section) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	var b strings.Builder
	line := strings.Repeat("=", 76)
	b.WriteString("\n" + w.paint("36", line) + "\n")
	b.WriteString(w.paint("1;36", Clean(title)) + "\n")
	b.WriteString("Started " + w.now().Format("2006-01-02 15:04:05 -07:00") + "\n")
	for _, section := range sections {
		b.WriteString("\n" + w.paint("1", Clean(section.Title)) + "\n")
		for _, row := range section.Rows {
			for _, part := range strings.Split(Clean(row), "\n") {
				b.WriteString("  " + part + "\n")
			}
		}
	}
	b.WriteString(w.paint("36", line) + "\n\n")
	n, err := io.WriteString(w.out, b.String())
	if err == nil && n != b.Len() {
		err = io.ErrShortWrite
	}
	return err
}

type AccessURL struct{ Label, URL string }

// AccessURLs reflects actual bind hosts. Wildcards advertise loopback and only
// private LAN candidates, without pretending to know which adapter a phone uses.
// The caller supplies the resolved HTTP or HTTPS port after server startup.
func AccessURLs(hosts, localIPs []string, scheme, port string) []AccessURL {
	if len(hosts) == 0 {
		hosts = []string{""}
	}
	locals := append([]string(nil), localIPs...)
	sort.Strings(locals)
	var out []AccessURL
	seen := map[string]bool{}
	add := func(host, label string) {
		u := (&url.URL{Scheme: scheme, Host: net.JoinHostPort(host, port), Path: "/"}).String()
		if !seen[u] {
			seen[u] = true
			out = append(out, AccessURL{label, u})
		}
	}
	for _, host := range hosts {
		switch host {
		case "", "0.0.0.0", "::":
			loopback := "127.0.0.1"
			if host == "::" {
				loopback = "::1"
			}
			add(loopback, "Web UI")
			count := 0
			for _, local := range locals {
				ip := net.ParseIP(local)
				if ip == nil || !ip.IsPrivate() || ip.IsLoopback() {
					continue
				}
				if host == "0.0.0.0" && ip.To4() == nil || host == "::" && ip.To4() != nil {
					continue
				}
				if count < 4 {
					add(local, "LAN candidate")
					count++
				}
			}
		default:
			add(host, "Web UI")
		}
	}
	return out
}

func Bytes(n uint64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	v := float64(n) / 1024
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f %s", v, units[i])
}
