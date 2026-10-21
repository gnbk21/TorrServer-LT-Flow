package utils

import (
	"reflect"
	"sort"
	"testing"
)

func TestEpisodeOrderIncludesEntireSharedNumber(t *testing.T) {
	files := []string{"Episode 10.mkv", "Episode 2.mkv", "Episode 1.mkv", "Episode 11.mkv", "Episode 02.mkv"}
	sort.Slice(files, func(i, j int) bool { return CompareStrings(files[i], files[j]) })
	want := []string{"Episode 1.mkv", "Episode 02.mkv", "Episode 2.mkv", "Episode 10.mkv", "Episode 11.mkv"}
	if !reflect.DeepEqual(files, want) {
		t.Fatal(files)
	}
	for _, file := range files {
		if CompareStrings(file, file) {
			t.Fatal("comparison is not strict")
		}
	}
}
