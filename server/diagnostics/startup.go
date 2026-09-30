package diagnostics

import (
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}
type StartupStatus struct {
	StartedAt       time.Time `json:"started_at"`
	ListenersReady  bool      `json:"listeners_ready"`
	ListenerReadyMs int64     `json:"listener_ready_ms"`
	EngineReady     bool      `json:"engine_ready"`
	EngineReadyMs   int64     `json:"engine_ready_ms"`
}

var startupMu sync.Mutex
var startupStatus = StartupStatus{StartedAt: time.Now()}

func MarkListenersReady(ready bool) {
	startupMu.Lock()
	startupStatus.ListenersReady = ready
	if ready {
		startupStatus.ListenerReadyMs = time.Since(startupStatus.StartedAt).Milliseconds()
	}
	startupMu.Unlock()
}
func MarkEngineReady(ready bool) {
	startupMu.Lock()
	startupStatus.EngineReady = ready
	if ready {
		startupStatus.EngineReadyMs = time.Since(startupStatus.StartedAt).Milliseconds()
	}
	startupMu.Unlock()
}
func Startup() StartupStatus { startupMu.Lock(); defer startupMu.Unlock(); return startupStatus }

// Doctor checks local prerequisites without a database, engine or external DNS.
// Account values and private filesystem contents are never printed.
func Doctor(stateDir, port string, ips []string, auth bool) ([]Check, bool) {
	checks := []Check{}
	valid := true
	add := func(name, status, detail string) {
		checks = append(checks, Check{name, status, detail})
		if status == "error" {
			valid = false
		}
	}
	if info, err := os.Stat(stateDir); err != nil || !info.IsDir() {
		add("state_directory", "error", "State directory is missing or inaccessible")
	} else {
		add("state_directory", "ok", "State directory exists")
	}
	if auth {
		buf, err := os.ReadFile(filepath.Join(stateDir, "accs.db"))
		accounts := map[string]string{}
		if err == nil {
			err = json.Unmarshal(buf, &accounts)
		}
		good := err == nil && len(accounts) > 0
		for name, password := range accounts {
			if name == "" || password == "" {
				good = false
			}
		}
		if !good {
			add("authentication", "error", "--httpauth requires a valid nonempty accs.db account map")
		} else {
			add("authentication", "ok", "Configured account map is valid")
		}
	} else {
		add("authentication", "notice", "HTTP authentication is disabled")
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		add("http_port", "error", "Port must be between 1 and 65535")
	} else {
		if len(ips) == 0 {
			ips = []string{""}
		}
		listeners := []net.Listener{}
		for _, ip := range ips {
			listener, err := net.Listen("tcp", net.JoinHostPort(ip, port))
			if err != nil {
				add("http_listener", "error", "Cannot bind the configured address and port")
			} else {
				listeners = append(listeners, listener)
				add("http_listener", "ok", "Configured address can be bound")
			}
		}
		for _, listener := range listeners {
			listener.Close()
		}
	}
	for _, tool := range []string{"ffprobe", "gst-launch-1.0"} {
		if _, err := exec.LookPath(tool); err != nil {
			add(tool, "optional_missing", "Optional dependency is not on PATH; external-player playback remains available")
		} else {
			add(tool, "ok", "Optional dependency is available on PATH")
		}
	}
	return checks, valid
}
