package web

import (
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestListenerShutdownStopsAccepting(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := probe.Addr().String()
	probe.Close()
	if err := startListener(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), addr, false); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(shutdownListeners)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	shutdownListeners()
	conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
	if err == nil {
		conn.Close()
		t.Fatal("listener still accepts after shutdown")
	}
}
