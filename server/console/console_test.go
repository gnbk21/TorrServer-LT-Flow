package console

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPlainLogSeverityAndMetadataSafety(t *testing.T) {
	var b bytes.Buffer
	w := New(&b, false)
	w.now = func() time.Time { return time.Date(2026, 10, 1, 12, 34, 56, 0, time.UTC) }
	for _, message := range []string{
		"No error occurred\n", "Error opening file\n", "[WARN] [Network] retry\n",
		"[INFO] [Status] first\nsecond\n", "title\x1b[2J\x1b]0;fake title\x07\r\u202eX\n",
	} {
		if n, err := w.Write([]byte(message)); err != nil || n != len(message) {
			t.Fatalf("Write: %d %v", n, err)
		}
	}
	got := b.String()
	for _, want := range []string{"12:34:56 INFO  No error occurred", "12:34:56 ERROR Error opening file", "12:34:56 WARN  [Network] retry", "12:34:56 INFO  second", "title??X"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
	if strings.ContainsAny(got, "\x1b\r\u202e\x07") {
		t.Fatal("untrusted terminal controls survived")
	}
}

func TestColorResetAndPanelAtomicity(t *testing.T) {
	var b bytes.Buffer
	w := New(&b, true)
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = w.Write([]byte("[ERROR] [Server] failed\n")) }()
	}
	if err := w.Panel("FLOW", []Section{{"CONNECTION", []string{"http://127.0.0.1:8090/", "row\r\x1b[2J"}}}); err != nil {
		t.Fatal(err)
	}
	wg.Wait()
	got := b.String()
	if !strings.Contains(got, "\x1b[31mERROR\x1b[0m") {
		t.Fatal("missing error color/reset")
	}
	start, end := strings.Index(got, "FLOW"), strings.Index(got, "row?")
	if start < 0 || end < start || strings.Contains(got[start:end], "failed") {
		t.Fatalf("panel interleaved: %q", got)
	}
	if strings.Contains(got, "\x1b[2J") {
		t.Fatal("panel contained screen clearing command")
	}
}

type failingWriter struct{ short bool }

func (w failingWriter) Write(p []byte) (int, error) {
	if w.short {
		return len(p) - 1, nil
	}
	return 0, errors.New("output closed")
}

func TestOutputFailures(t *testing.T) {
	for _, short := range []bool{false, true} {
		w := New(failingWriter{short}, false)
		if _, err := w.Write([]byte("test")); err == nil {
			t.Fatal("lost write error")
		}
		if err := w.Panel("title", nil); err == nil {
			t.Fatal("lost panel error")
		}
	}
	_, err := New(failingWriter{true}, false).Write([]byte("test"))
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
}

func TestAccessURLsBindCompatibility(t *testing.T) {
	tests := []struct {
		hosts, locals []string
		scheme, port  string
		want          []AccessURL
	}{
		{[]string{"127.0.0.1"}, []string{"192.168.1.7"}, "http", "8090", []AccessURL{{"Web UI", "http://127.0.0.1:8090/"}}},
		{[]string{"::1", "::1"}, nil, "https", "8091", []AccessURL{{"Web UI", "https://[::1]:8091/"}}},
		{nil, []string{"8.8.8.8", "192.168.1.7", "127.0.0.1", "fe80::1", "fd00::2"}, "http", "8090", []AccessURL{{"Web UI", "http://127.0.0.1:8090/"}, {"LAN candidate", "http://192.168.1.7:8090/"}, {"LAN candidate", "http://[fd00::2]:8090/"}}},
		{[]string{"0.0.0.0"}, []string{"fd00::2", "10.0.0.7"}, "https", "12345", []AccessURL{{"Web UI", "https://127.0.0.1:12345/"}, {"LAN candidate", "https://10.0.0.7:12345/"}}},
		{[]string{"::"}, []string{"10.0.0.7", "fd00::2"}, "https", "8091", []AccessURL{{"Web UI", "https://[::1]:8091/"}, {"LAN candidate", "https://[fd00::2]:8091/"}}},
		{[]string{"192.168.1.7", "localhost"}, nil, "http", "9000", []AccessURL{{"Web UI", "http://192.168.1.7:9000/"}, {"Web UI", "http://localhost:9000/"}}},
	}
	for _, tc := range tests {
		got := AccessURLs(tc.hosts, tc.locals, tc.scheme, tc.port)
		if len(got) != len(tc.want) {
			t.Fatalf("%v: got %v want %v", tc.hosts, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		}
	}
}

func TestCandidateLimitAndUnits(t *testing.T) {
	got := AccessURLs(nil, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4", "10.0.0.5"}, "http", "8090")
	if len(got) != 5 {
		t.Fatal(got)
	}
	for value, want := range map[uint64]string{0: "0 B", 1024: "1.0 KiB", 64 << 20: "64.0 MiB", 1 << 30: "1.0 GiB", ^uint64(0): "16.0 EiB"} {
		if got := Bytes(value); got != want {
			t.Fatalf("%d: %s != %s", value, got, want)
		}
	}
}

func TestOptionsAndColorFallback(t *testing.T) {
	for _, mode := range []string{"auto", "plain", "off"} {
		for _, interval := range []int{0, 5, 30, 3600} {
			if err := Validate(mode, interval); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, value := range []int{-1, 1, 4, 3601} {
		if Validate("auto", value) == nil {
			t.Fatal(value)
		}
	}
	if Validate("unknown", 30) == nil {
		t.Fatal("unknown mode accepted")
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	defer w.Close()
	for _, mode := range []string{"auto", "plain", "off"} {
		color, restore := ConfigureColor(w, mode)
		restore()
		if color {
			t.Fatal("ANSI enabled on pipe")
		}
	}
	t.Setenv("NO_COLOR", "1")
	color, restore := ConfigureColor(os.Stdout, "auto")
	restore()
	if color {
		t.Fatal("NO_COLOR ignored")
	}
}

func TestStatusPolicyAndCancellation(t *testing.T) {
	now := time.Now()
	if !ShouldReport(now, time.Time{}, 30*time.Second, "idle", "") || !ShouldReport(now, now, 30*time.Second, "active", "idle") {
		t.Fatal("missing initial/change report")
	}
	if ShouldReport(now.Add(29*time.Second), now, 30*time.Second, "idle", "idle") || ShouldReport(now, time.Time{}, 0, "idle", "") {
		t.Fatal("unexpected report")
	}
	if !ShouldReport(now.Add(30*time.Second), now, 30*time.Second, "idle", "idle") {
		t.Fatal("missing heartbeat")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	count := 0
	go func() {
		defer close(done)
		RunStatus(ctx, 30*time.Second, func() string { return "idle" }, func() string { return "idle" }, func(string) { count++; cancel() })
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel did not stop reporter")
	}
	if count != 1 {
		t.Fatal(count)
	}
	RunStatus(context.Background(), 0, func() string { t.Fatal("disabled reporter sampled"); return "" }, func() string { t.Fatal("disabled reporter formatted"); return "" }, func(string) { t.Fatal("disabled reporter emitted") })
}
