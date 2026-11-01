package diagnostics

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/trace"
	"testing"
	"testing/synctest"
	"time"
)

func TestIncidentCooldownPolicy(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		if !incidentDue(true, false, false, time.Time{}, start) {
			t.Fatal("initial admission")
		}
		for _, flags := range [][3]bool{{false, false, false}, {true, true, false}, {true, false, true}} {
			if incidentDue(flags[0], flags[1], flags[2], time.Time{}, start) {
				t.Fatal("unsafe admission")
			}
		}
		time.Sleep(IncidentCooldown - time.Nanosecond)
		if incidentDue(true, false, false, start, time.Now()) {
			t.Fatal("early retry")
		}
		time.Sleep(time.Nanosecond)
		if !incidentDue(true, false, false, start, time.Now()) {
			t.Fatal("cooldown not recovered")
		}
	})
}
func TestIncidentRetentionAndIncompleteDiscard(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 4; i++ {
		err := writeIncident(dir, func(w io.Writer) (int64, error) {
			n, e := w.Write([]byte("complete trace fixture"))
			return int64(n), e
		}, time.Unix(int64(i+1), 0))
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(incidentFiles(dir)) != 2 {
		t.Fatal("retention")
	}
	err := writeIncident(dir, func(w io.Writer) (int64, error) {
		block := make([]byte, 1<<20)
		var total int64
		for i := 0; i < 65; i++ {
			n, e := w.Write(block)
			total += int64(n)
			if e != nil {
				return total, e
			}
		}
		return total, nil
	}, time.Now())
	if err == nil || len(incidentFiles(dir)) != 2 {
		t.Fatal("oversized capture retained")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatal("partial capture remains")
	}
	r := IncidentRecorder{dir: dir}
	for _, name := range []string{"../secret", "incident-../secret.trace", "settings.json"} {
		if f, e := r.OpenRaw(name); e == nil {
			f.Close()
			t.Fatal("path admitted")
		}
	}
	f, e := r.OpenRaw(incidentFiles(dir)[0].Name)
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
}
func TestRealFlightRecorderProducesTrace(t *testing.T) {
	dir := t.TempDir()
	r := trace.NewFlightRecorder(trace.FlightRecorderConfig{MinAge: 30 * time.Second, MaxBytes: 32 << 20})
	if err := r.Start(); err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	for i := 0; i < 30; i++ {
		runtime.Gosched()
	}
	if err := writeIncident(dir, r.WriteTo, time.Now()); err != nil {
		t.Fatal(err)
	}
	files := incidentFiles(dir)
	if len(files) != 1 {
		t.Fatal(files)
	}
	data, err := os.ReadFile(filepath.Join(dir, files[0].Name))
	if err != nil || len(data) < 16 || string(data[:3]) != "go " {
		t.Fatal("invalid trace header", err)
	}
}

func TestIncidentRestartCleanupAndPressure(t *testing.T) {
	base := t.TempDir()
	r := &IncidentRecorder{}
	r.Configure(base, false, false)
	if _, err := os.Stat(filepath.Join(base, "flow-incidents")); !os.IsNotExist(err) {
		t.Fatal("disabled recorder created storage")
	}
	dir, err := incidentDirectory(base, true)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".incident-owned.partial", "unrelated.partial", "incident-retained.trace"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	r.Configure(base, false, false)
	if len(r.Status().Files) != 1 {
		t.Fatal("disabled restart hid retained capture")
	}
	r.Configure(base, true, false)
	t.Cleanup(func() { r.Configure(base, false, false) })
	if !r.Status().Recording {
		t.Fatal(r.Status())
	}
	if _, err := os.Stat(filepath.Join(dir, ".incident-owned.partial")); !os.IsNotExist(err) {
		t.Fatal("incomplete capture survived")
	}
	if _, err := os.Stat(filepath.Join(dir, "unrelated.partial")); err != nil {
		t.Fatal("unrelated file removed")
	}
	r.Configure(base, true, true)
	if s := r.Status(); s.Recording || !s.PressurePaused || r.Capture("SERVER_READ_STALL") {
		t.Fatal("recorder continued under pressure", s)
	}
	r.Configure(base, true, false)
	if !r.Status().Recording {
		t.Fatal("recorder did not recover")
	}
}
