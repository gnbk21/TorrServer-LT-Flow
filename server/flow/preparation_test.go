package flow

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreparationQuotaUnionAndBoundary(t *testing.T) {
	h := "0123456789012345678901234567890123456789"
	rs := []PieceReservation{{h, 0, 2, 16, 53}, {h, 2, 3, 16, 53}}
	n, err := PreparationReserved(rs)
	if err != nil || n != 53 {
		t.Fatalf("union %d %v", n, err)
	}
	rs[1].PieceLength = 8
	if _, err = PreparationReserved(rs); err == nil {
		t.Fatal("accepted inconsistent geometry")
	}
	if _, err = PreparationReserved([]PieceReservation{{h, 0, 10, 16, 53}}); err == nil {
		t.Fatal("accepted invalid piece index")
	}
}
func TestPreparationAtomicRecovery(t *testing.T) {
	name := filepath.Join(t.TempDir(), "jobs.json")
	want := map[string]string{"state": "paused"}
	if err := WritePreparationState(name, want); err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := ReadPreparationState(name, &got); err != nil || got["state"] != "paused" {
		t.Fatalf("%v %v", got, err)
	}
	if err := WritePreparationState(name, map[string]string{"state": "ready"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(`{"state":`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ReadPreparationState(name, &got); err == nil {
		t.Fatal("accepted interrupted state")
	}
}
