package torrstor

import (
	"reflect"
	"testing"
	"time"
)

func TestRecoveryOwnersMergeAndRetireIndependently(t *testing.T) {
	var f recoveryFrontiers
	now := time.Unix(100, 0)
	f.update([]int{4, 8}, true, now, 100)
	if got := f.update([]int{20, 4}, false, now, 100); !reflect.DeepEqual(got, []int{20, 4, 8}) {
		t.Fatalf("reader and bootstrap demand: %v", got)
	}
	if got := f.update(nil, true, now, 100); !reflect.DeepEqual(got, []int{20, 4}) {
		t.Fatalf("retiring bootstrap erased reader demand: %v", got)
	}
	if got := f.update([]int{9}, true, now.Add(4*time.Second), 100); !reflect.DeepEqual(got, []int{9}) {
		t.Fatalf("preload revived expired reader: %v", got)
	}
	if got := f.update(nil, false, now.Add(8*time.Second), 100); len(got) != 0 {
		t.Fatalf("reader update revived expired preload: %v", got)
	}
}

func TestRecoveryFrontiersAreBoundedCopiedAndValidatePieces(t *testing.T) {
	var f recoveryFrontiers
	now := time.Unix(100, 0)
	input := []int{-1, 30, 0, 1, 2, 2, 3, 4, 5, 6, 7, 8, 9}
	got := f.update(input, true, now, 30)
	if !reflect.DeepEqual(got, []int{0, 1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("unbounded/invalid recovery demand: %v", got)
	}
	input[2], got[0] = 29, 29
	got = f.update([]int{25}, false, now, 30)
	if !reflect.DeepEqual(got, []int{25, 0, 1, 2, 3, 4, 5, 6}) {
		t.Fatalf("caller mutated lease or reader lost priority: %v", got)
	}
	if got := f.update(nil, false, now.Add(-time.Second), 30); len(got) != 0 {
		t.Fatalf("clock reversal revived future preload: %v", got)
	}
}
