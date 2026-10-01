package log

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestInitSharedLogPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "combined.log")

	Init(path, path)
	defer Close()

	if logFile == nil {
		t.Fatal("expected logFile to be set")
	}
	if webLogFile != logFile {
		t.Fatal("expected webLogFile to share logFile when paths are equal")
	}
	if webLog == nil {
		t.Fatal("expected webLog to be set")
	}
}

func TestInitSeparateLogPaths(t *testing.T) {
	dir := t.TempDir()
	serverPath := filepath.Join(dir, "server.log")
	webPath := filepath.Join(dir, "web.log")

	Init(serverPath, webPath)
	defer Close()

	if logFile == nil || webLogFile == nil {
		t.Fatal("expected both log files to be set")
	}
	if logFile == webLogFile {
		t.Fatal("expected separate file handles for different paths")
	}
}

func TestCloseSharedLogPathOnce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "combined.log")

	Init(path, path)
	Close()

	if logFile != nil || webLogFile != nil || webLog != nil {
		t.Fatal("expected log handles to be cleared after Close")
	}
}

func TestConsoleStopOnce(t *testing.T) {
	count := 0
	SetConsoleStop(func() { count++ })
	StopConsoleStatus()
	StopConsoleStatus()
	Close()
	if count != 1 {
		t.Fatalf("reporter stopped %d times", count)
	}
}

func TestConcurrentConsoleStopJoins(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var joined atomic.Bool
	SetConsoleStop(func() { close(started); <-release; joined.Store(true) })
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			StopConsoleStatus()
			if !joined.Load() {
				t.Error("stop returned before reporter joined")
			}
		}()
	}
	<-started
	close(release)
	wg.Wait()
}

func TestConcurrentConsoleCloseRestoresOnce(t *testing.T) {
	var restored atomic.Int32
	restoreConsole = func() { restored.Add(1) }
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); Close() }()
	}
	wg.Wait()
	if restored.Load() != 1 {
		t.Fatalf("terminal restored %d times", restored.Load())
	}
}
