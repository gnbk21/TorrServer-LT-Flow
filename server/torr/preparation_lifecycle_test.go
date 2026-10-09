package torr

import (
	"os"
	"path/filepath"
	"server/settings"
	"testing"
)

func TestPreparationCleanupCannotBeReversed(t *testing.T) {
	previous, readOnly := helperEngine(), settings.ReadOnly
	settings.ReadOnly = false
	t.Cleanup(func() { InitApiHelper(previous); settings.ReadOnly = readOnly })
	hash := NewHashFromHex("0123456789012345678901234567890123456789")
	id := preparationID(hash, 1)
	job := &preparationRecord{PreparationJob: PreparationJob{ID: id, State: "cleaning"}}
	InitApiHelper(&BTServer{preparation: &preparationManager{jobs: map[string]*preparationRecord{id: job}}})
	for _, action := range []string{"start", "resume", "pause", "cancel"} {
		if err := PrepareEpisode(hash.HexString(), 1, action); err == nil {
			t.Fatalf("%s reversed a native cleanup fence", action)
		}
		if job.State != "cleaning" {
			t.Fatal("cleanup state changed")
		}
	}
	if err := PrepareEpisode(hash.HexString(), 1, "remove"); err != nil {
		t.Fatal("repeated cleanup must be idempotent", err)
	}
}

func TestPreparationActionSaveFailurePreservesScheduling(t *testing.T) {
	previous, readOnly := helperEngine(), settings.ReadOnly
	settings.ReadOnly = false
	t.Cleanup(func() { InitApiHelper(previous); settings.ReadOnly = readOnly })
	hash := NewHashFromHex("0123456789012345678901234567890123456789")
	id := preparationID(hash, 1)
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	job := &preparationRecord{PreparationJob: PreparationJob{ID: id, State: "paused", ErrorCode: "previous", SchedulingReason: "PLAYBACK_PRIORITY"}}
	InitApiHelper(&BTServer{preparation: &preparationManager{name: filepath.Join(blocked, "jobs.json"), root: root, jobs: map[string]*preparationRecord{id: job}}})
	if err := PrepareEpisode(hash.HexString(), 1, "resume"); err == nil {
		t.Fatal("failed write was accepted")
	}
	if job.State != "paused" || job.ErrorCode != "previous" || job.SchedulingReason != "PLAYBACK_PRIORITY" {
		t.Fatal("failed action changed scheduling", job.PreparationJob)
	}
}

func TestLibraryDeletionSchedulesAllPreparationJobsAndRollsBackSaveFailure(t *testing.T) {
	previous := settings.ReadOnly
	settings.ReadOnly = false
	t.Cleanup(func() { settings.ReadOnly = previous })
	hash := NewHashFromHex("1234567890123456789012345678901234567890")
	other := NewHashFromHex("2234567890123456789012345678901234567890")
	root := t.TempDir()
	first := &preparationRecord{PreparationJob: PreparationJob{ID: preparationID(hash, 1), State: "downloading"}, Spec: TorrentSpec{InfoHash: hash}}
	second := &preparationRecord{PreparationJob: PreparationJob{ID: preparationID(hash, 2), State: "ready"}, Spec: TorrentSpec{InfoHash: hash}}
	untouched := &preparationRecord{PreparationJob: PreparationJob{ID: preparationID(other, 1), State: "paused"}, Spec: TorrentSpec{InfoHash: other}}
	p := &preparationManager{name: filepath.Join(root, "jobs.json"), root: root, jobs: map[string]*preparationRecord{first.ID: first, second.ID: second, untouched.ID: untouched}}
	if handled, err := p.removeTorrentJobs(hash); !handled || err != nil {
		t.Fatal(handled, err)
	}
	if first.State != "cleaning" || second.State != "cleaning" || untouched.State != "paused" {
		t.Fatal("library deletion did not isolate its preparation cleanup")
	}
	if err := os.WriteFile(filepath.Join(root, "blocked"), []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	p.name = filepath.Join(root, "blocked", "jobs.json")
	first.State, second.State = "downloading", "ready"
	if handled, err := p.removeTorrentJobs(hash); !handled || err == nil {
		t.Fatal("failed state write was accepted", handled, err)
	}
	if first.State != "downloading" || second.State != "ready" {
		t.Fatal("failed deletion changed preparation demand")
	}
}
