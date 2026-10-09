//go:build gst

package gstreamer

import "testing"

func TestCueToleranceIncludesOneFrameAndRejectsInvalidRates(t *testing.T) {
	for _, tc := range []struct {
		num, den int
		want     uint64
	}{{25, 1, 41_000_000}, {30000, 1001, 34_366_667}, {0, 1, 1_000_000}, {25, 0, 1_000_000}, {1, 2, 1_000_000}, {-1, 1, 1_000_000}} {
		task := &Task{Cue: &CueTimeline{TimestampScaleNS: 1_000_000}, Probe: ProbeInfo{Tracks: []TrackInfo{{Type: "video", FrameRateNum: tc.num, FrameRateDen: tc.den}}}}
		if got := cueBoundaryToleranceNS(task); got != tc.want {
			t.Fatalf("%d/%d: %d, expected %d", tc.num, tc.den, got, tc.want)
		}
	}
	if cueBoundaryToleranceNS(nil) != 1 {
		t.Fatal("nil task tolerance")
	}
}

func TestCueParserAcceptsFrameDriftButRejectsLargeMismatch(t *testing.T) {
	task := &Task{Cue: &CueTimeline{TimestampScaleNS: 1_000_000}, Probe: ProbeInfo{Tracks: []TrackInfo{{Type: "video", FrameRateNum: 25, FrameRateDen: 1}}}}
	for _, boundary := range []uint64{2960, 3040, 3100} {
		r := &mp4BoxReader{cueMode: true, videoTrack: trackInfo{timescale: 1000}, video: []mp4Fragment{testFragment(1, 1000, 0, 40, 4, true, 1), testFragment(1, 1000, boundary, 40, 4, true, 2)}}
		if err := r.SetTargetSegment(0, 3_000_000_000, cueBoundaryToleranceNS(task)); err != nil {
			t.Fatal(err)
		}
		count, err := r.selectCueVideoCount()
		if boundary == 3100 {
			if err == nil {
				t.Fatal("large mismatch accepted")
			}
		} else if err != nil || count != 1 {
			t.Fatal("one-frame drift rejected", boundary, count, err)
		}
	}
}
