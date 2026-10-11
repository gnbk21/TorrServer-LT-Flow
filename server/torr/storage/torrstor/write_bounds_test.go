package torrstor

import (
	"bytes"
	"math"
	"testing"
)

func FuzzStorageWriteBounds(f *testing.F) {
	f.Add(int64(0), []byte("valid"))
	f.Add(int64(-1), []byte("invalid"))
	f.Add(int64(math.MaxInt64), []byte("overflow"))
	f.Add(int64(65535), []byte("crossing"))
	f.Add(int64(65536), []byte{})
	f.Fuzz(func(t *testing.T, offset int64, payload []byte) {
		if len(payload) > 65536 {
			t.Skip()
		}
		s := NewStorage()
		s.callbackOpen(1, mkHash(0x74), 2, 65536)
		s.callbackSize(1, 131072)
		t.Cleanup(func() { s.callbackClose(1) })
		n, err := s.callbackWrite(1, 0, offset, payload)
		valid := offset >= 0 && offset <= 65536 && int64(len(payload)) <= 65536-offset
		if !valid {
			if n != 0 || err != errOutOfPiece {
				t.Fatalf("invalid write: n=%d error=%v", n, err)
			}
			if s.lookup(1).Filled() != 0 {
				t.Fatal("invalid write grew resident data")
			}
			return
		}
		if n != len(payload) || err != nil {
			t.Fatalf("valid write: n=%d error=%v", n, err)
		}
		if len(payload) == 0 {
			return
		}
		got := make([]byte, len(payload))
		if n, err := s.callbackRead(1, 0, offset, got); n != len(got) || err != nil || !bytes.Equal(got, payload) {
			t.Fatalf("valid write did not preserve bytes: n=%d error=%v", n, err)
		}
	})
}
