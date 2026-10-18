package torr

import (
	"reflect"
	storageState "server/torr/storage/state"
	"testing"
)

func TestBootstrapDoesNotUrgentlyRequestLaterHead(t *testing.T) {
	// Three 16-MiB gate pieces previously caused the straggler loop to promote
	// phase-2 piece 129 while bootstrap still needed 128 and EOF piece 7899.
	snapshot := map[int]storageState.ItemState{}
	got := preloadStragglers(snapshot, []int{128, 7899}, 7899, 7899, map[int]bool{})
	if !reflect.DeepEqual(got, []int{7899, 128}) {
		t.Fatalf("bootstrap ordering: %v", got)
	}
	assigned := map[int]bool{128: true, 7899: true}
	if got := preloadStragglers(snapshot, []int{128, 7899}, 7899, 7899, assigned); len(got) != 0 {
		t.Fatalf("existing deadlines would be reset: %v", got)
	}
	snapshot[128], snapshot[7899] = storageState.ItemState{Completed: true}, storageState.ItemState{Completed: true}
	got = preloadStragglers(snapshot, []int{128, 129, 130, 7899}, 7899, 7899, assigned)
	if !reflect.DeepEqual(got, []int{129, 130}) {
		t.Fatalf("phase-2 demand missing: %v", got)
	}
}

func TestPreloadStragglersStayBoundedAndSkipVerifiedPieces(t *testing.T) {
	pieces := []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
	if got := preloadStragglers(nil, pieces, 8, 8, nil); len(got) != 0 {
		t.Fatal("bulk preload became time critical")
	}
	snapshot := map[int]storageState.ItemState{0: {Completed: true}}
	got := preloadStragglers(snapshot, pieces, 8, 8, map[int]bool{1: true})
	if len(got) != 7 || got[0] != 8 {
		t.Fatalf("bounded tail-first stragglers: %v", got)
	}
}
