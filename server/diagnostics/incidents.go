package diagnostics

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
	"time"
)

const IncidentOutputLimit int64 = 64 << 20
const IncidentCooldown = 5 * time.Minute

type IncidentFile struct {
	Name      string    `json:"name"`
	Bytes     int64     `json:"bytes"`
	CreatedAt time.Time `json:"created_at"`
}
type IncidentStatus struct {
	Enabled        bool           `json:"enabled"`
	Recording      bool           `json:"recording"`
	Capturing      bool           `json:"capturing"`
	PressurePaused bool           `json:"pressure_paused"`
	Code           string         `json:"code"`
	LastReason     string         `json:"last_reason,omitempty"`
	LastAttempt    time.Time      `json:"last_attempt"`
	Files          []IncidentFile `json:"files"`
}

// The ring's 32 MiB size is a runtime hint, not a memory guarantee. Raw output
// has its own hard bound and is never part of the ordinary support report.
type IncidentRecorder struct {
	mu       sync.Mutex
	recorder *trace.FlightRecorder
	dir      string
	status   IncidentStatus
}

var Incidents IncidentRecorder

func (r *IncidentRecorder) Configure(base string, enabled, pressure bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// Previously captured files remain selectable after a restart even when the
	// recorder is disabled. Merely listing them never creates a directory.
	if r.dir == "" {
		if dir, err := incidentDirectory(base, false); err == nil {
			r.dir = dir
		}
	}
	r.status.Enabled, r.status.PressurePaused = enabled, pressure
	if r.status.Capturing {
		return
	} // completion applies a pending stop
	if !enabled || pressure {
		if r.recorder != nil {
			r.recorder.Stop()
			r.recorder = nil
		}
		r.status.Recording = false
		if pressure {
			r.status.Code = "PRESSURE_PAUSED"
		} else {
			r.status.Code = "DISABLED"
		}
		return
	}
	if r.recorder != nil {
		return
	}
	dir, err := incidentDirectory(base, true)
	if err != nil {
		r.status.Code = "STORAGE_UNAVAILABLE"
		return
	}
	if err := cleanIncidentPartials(dir); err != nil {
		r.status.Code = "STORAGE_UNAVAILABLE"
		return
	}
	r.dir = dir
	recorder := trace.NewFlightRecorder(trace.FlightRecorderConfig{MinAge: 30 * time.Second, MaxBytes: 32 << 20})
	if err := recorder.Start(); err != nil {
		r.status.Code = "RECORDER_UNAVAILABLE"
		return
	}
	r.recorder = recorder
	r.status.Recording, r.status.Code = true, "RECORDING"
}

func incidentDirectory(base string, create bool) (string, error) {
	root, err := filepath.EvalSymlinks(base)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "flow-incidents")
	if create {
		err = os.Mkdir(dir, 0700)
	}
	if err != nil && !os.IsExist(err) {
		return "", err
	}
	st, err := os.Lstat(dir)
	if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("incident directory must be a real directory")
	}
	return dir, nil
}

// Only recorder-owned incomplete regular files are discarded. Symlinks and
// unrelated files are never followed or deleted, including after a crash.
func cleanIncidentPartials(dir string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	defer f.Close()
	entries, err := f.ReadDir(257)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if len(entries) > 256 {
		return errors.New("incident directory inventory limit exceeded")
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".incident-") && strings.HasSuffix(entry.Name(), ".partial") && entry.Type().IsRegular() {
			if err := root.Remove(entry.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}

func incidentDue(enabled, pressure, busy bool, last, now time.Time) bool {
	return enabled && !pressure && !busy && (last.IsZero() || now.Sub(last) >= IncidentCooldown)
}

// Capture admits one asynchronous write. No HTTP operation or player command
// is retried, and the incident has no media/client identity or arbitrary text.
func (r *IncidentRecorder) Capture(reason string) bool {
	if reason != "SERVER_READ_STALL" && reason != "STARTUP_FAILURE" {
		return false
	}
	r.mu.Lock()
	now := time.Now()
	if r.recorder == nil || !incidentDue(r.status.Enabled, r.status.PressurePaused, r.status.Capturing, r.status.LastAttempt, now) {
		r.mu.Unlock()
		return false
	}
	r.status.Capturing, r.status.LastAttempt, r.status.LastReason = true, now, reason
	recorder, dir := r.recorder, r.dir
	r.mu.Unlock()
	go func() {
		err := writeIncident(dir, recorder.WriteTo, now)
		r.mu.Lock()
		defer r.mu.Unlock()
		r.status.Capturing = false
		if err != nil {
			r.status.Code = "CAPTURE_DISCARDED"
		} else {
			r.status.Code = "CAPTURE_SAVED"
		}
		if !r.status.Enabled || r.status.PressurePaused {
			recorder.Stop()
			r.recorder = nil
			r.status.Recording = false
		}
	}()
	return true
}

type incidentWriter struct {
	io.Writer
	remaining int64
}

func (w *incidentWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		return 0, errors.New("incident output exceeds hard limit")
	}
	n, err := w.Writer.Write(p)
	w.remaining -= int64(n)
	return n, err
}
func writeIncident(dir string, write func(io.Writer) (int64, error), now time.Time) error {
	f, err := os.CreateTemp(dir, ".incident-*.partial")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	defer f.Close()
	if err = RestrictPrivateFile(name); err != nil {
		return err
	}
	w := &incidentWriter{Writer: f, remaining: IncidentOutputLimit}
	n, err := write(w)
	if err != nil {
		return err
	}
	if n <= 0 || w.remaining == IncidentOutputLimit {
		return errors.New("empty incident")
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	files := incidentFiles(dir)
	// Make room first: a locked old capture prevents admission, rather than
	// exceeding retention or discarding a required deletion error.
	for _, old := range files[min(1, len(files)):] {
		if err = os.Remove(filepath.Join(dir, old.Name)); err != nil {
			return err
		}
	}
	final := filepath.Join(dir, fmt.Sprintf("incident-%s-%s.trace", now.UTC().Format("20060102T150405.000000000"), strings.TrimSuffix(strings.TrimPrefix(filepath.Base(name), ".incident-"), ".partial")))
	if err = os.Rename(name, final); err != nil {
		return err
	}
	return nil
}
func validIncidentName(name string) bool {
	return filepath.Base(name) == name && strings.HasPrefix(name, "incident-") && strings.HasSuffix(name, ".trace") && !strings.ContainsAny(name, `/\`)
}
func incidentFiles(dir string) []IncidentFile {
	out := []IncidentFile{}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if !validIncidentName(entry.Name()) || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		st, err := entry.Info()
		if err == nil && st.Mode().IsRegular() {
			out = append(out, IncidentFile{entry.Name(), st.Size(), st.ModTime().UTC()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name > out[j].Name })
	return out
}
func (r *IncidentRecorder) Status() IncidentStatus {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.status
	s.Files = incidentFiles(r.dir)
	return s
}

// OpenRaw requires an explicitly selected opaque filename, never a caller path.
func (r *IncidentRecorder) OpenRaw(name string) (*os.File, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !validIncidentName(name) || r.dir == "" {
		return nil, os.ErrNotExist
	}
	root, err := os.OpenRoot(r.dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	st, err := root.Lstat(name)
	if err != nil || !st.Mode().IsRegular() {
		return nil, os.ErrNotExist
	}
	return root.Open(name)
}
