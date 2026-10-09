//go:build gst

package gstreamer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"server/settings"
	"sync/atomic"
	"testing"
	"time"
)

// A subprocess fixture exercises the real discovery command, ownership and
// cancellation paths. It is not a substitute for the GStreamer media E2E gate.
func discovererFixture(t *testing.T, mode string) (Config, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fixture uses the native Linux CI Python runtime")
	}
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	// Use the installed runtime's real base library so this explicit root takes
	// precedence over /usr's real discoverer even during the media E2E gate.
	// A PATH-only command loses to a valid default runtime root.
	var base string
	for _, installed := range gstRuntimeRoots(Config{}) {
		for _, candidate := range gstBaseLibraryCandidates(installed) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				base = candidate
				break
			}
		}
		if base != "" {
			break
		}
	}
	if base == "" {
		t.Fatal("installed GStreamer base library is required by the lifecycle gate")
	}
	if err := os.Mkdir(filepath.Join(root, "lib"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(base, filepath.Join(root, "lib", "libgstreamer-1.0.so.0")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FLOW_GST_PROBE_FIXTURE", root)
	t.Setenv("FLOW_GST_PROBE_FIXTURE_MODE", mode)
	script := `#!/usr/bin/env python3
import os, pathlib, time
root=pathlib.Path(os.environ["FLOW_GST_PROBE_FIXTURE"])
with (root/"calls").open("a") as f: f.write("call\n")
mode=os.environ["FLOW_GST_PROBE_FIXTURE_MODE"]
if mode=="cold" and not (root/"warmed").exists(): raise SystemExit(0)
if mode=="wait":
    while not (root/"release").exists(): time.sleep(0.005)
print("Properties:\n  Duration: 0:01:00.000000000\n  container: Matroska\n    video #1: H.264\n      Frame rate: 25/1")
`
	if err := os.WriteFile(filepath.Join(root, "bin", gstDiscovererExecutableName()), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	conf := Config{GSTPath: root}
	if selected, err := gstDiscovererPath(conf); err != nil || selected != filepath.Join(root, "bin", gstDiscovererExecutableName()) {
		t.Fatalf("fixture discoverer was not selected: %q, %v", selected, err)
	}
	return conf, root
}

func waitProbeCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("probe lifecycle did not reach expected state")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestSharedProbeOneCallerCancelsWithoutAbortingAnother(t *testing.T) {
	conf, root := discovererFixture(t, "wait")
	s := NewService(conf)
	defer s.Dispose()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { _, err := s.ProbeContext(ctx, "fixture", "1"); first <- err }()
	waitProbeCondition(t, func() bool { _, err := os.Stat(filepath.Join(root, "calls")); return err == nil })
	go func() { _, err := s.ProbeContext(context.Background(), "fixture", "1"); second <- err }()
	waitProbeCondition(t, func() bool {
		s.probeMu.Lock()
		defer s.probeMu.Unlock()
		run := s.probeRuns[probeCacheKey("fixture", "1")]
		return run != nil && run.waiters == 2
	})
	cancel()
	select {
	case err := <-first:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("caller cancellation blocked")
	}
	if err := os.WriteFile(filepath.Join(root, "release"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-second:
		if err != nil {
			t.Fatal("second caller lost shared work", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("shared probe did not finish")
	}
	calls, _ := os.ReadFile(filepath.Join(root, "calls"))
	if string(calls) != "call\n" {
		t.Fatal("discovery was duplicated", string(calls))
	}
}

func TestDisposeCancelsInFlightDiscovery(t *testing.T) {
	conf, root := discovererFixture(t, "wait")
	s := NewService(conf)
	defer s.Dispose()
	done := make(chan error, 1)
	go func() { _, err := s.Probe("fixture", "1"); done <- err }()
	waitProbeCondition(t, func() bool { _, err := os.Stat(filepath.Join(root, "calls")); return err == nil })
	s.Dispose()
	select {
	case err := <-done:
		if !errors.Is(err, ErrServiceClosed) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("disposed probe kept waiting")
	}
}

func TestColdProbeWarmsOnceAndRetries(t *testing.T) {
	conf, root := discovererFixture(t, "cold")
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Range") != fmt.Sprintf("bytes=0-%d", probeWarmupHeadBytes-1) {
			t.Error("unexpected warmup range", r.Header.Get("Range"))
		}
		if err := os.WriteFile(filepath.Join(root, "warmed"), nil, 0600); err != nil {
			t.Error(err)
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", probeWarmupHeadBytes-1, probeWarmupHeadBytes))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(make([]byte, probeWarmupHeadBytes))
	}))
	defer srv.Close()
	settings.SetInternalBaseURL(srv.URL)
	defer settings.SetInternalBaseURL("")
	s := NewService(conf)
	defer s.Dispose()
	probe, err := s.Probe("fixture", "1")
	if err != nil || probe.Video() == nil {
		t.Fatal("cold retry failed", err)
	}
	calls, _ := os.ReadFile(filepath.Join(root, "calls"))
	if requests.Load() != 1 || string(calls) != "call\ncall\n" {
		t.Fatal("retry was not bounded", requests.Load(), string(calls))
	}
}

func TestWarmProbeCancellationStopsHTTPRead(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan bool, 1)
	go func() { done <- warmProbeSource(ctx, srv.URL, 0) }()
	<-started
	cancel()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("cancelled warmup succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled HTTP warmup blocked")
	}
}
