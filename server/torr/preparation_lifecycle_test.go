package torr

import (
	"server/settings"
	"testing"
)

func TestPreparationCleanupCannotBeReversed(t *testing.T) {
	previous, readOnly := bts, settings.ReadOnly
	settings.ReadOnly = false
	t.Cleanup(func() { bts, settings.ReadOnly = previous, readOnly })
	hash := NewHashFromHex("0123456789012345678901234567890123456789")
	id := preparationID(hash, 1)
	job := &preparationRecord{PreparationJob: PreparationJob{ID: id, State: "cleaning"}}
	bts = &BTServer{preparation: &preparationManager{jobs: map[string]*preparationRecord{id: job}}}
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
