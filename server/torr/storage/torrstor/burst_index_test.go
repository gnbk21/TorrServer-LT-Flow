package torrstor

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"server/flow"
	"server/settings"
)

func TestResidentMatroskaIndexReachesAdaptiveControllerWithoutFetching(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.SwarmProfile, f.ContainerBurstHints = "adaptive", true
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB, ReaderReadAHead: 95, Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(302, mkHash(0x32), 4096, pieceBlockSize)
	s.callbackSize(302, 64*flow.MiB)
	c := s.lookup(302)
	defer s.callbackClose(302)
	// Unknown-length Segment, 1 ms tick scale, three resident CuePoints. Their
	// cluster positions refer to future media; inspection must not fetch it.
	header := []byte{0x1a, 0x45, 0xdf, 0xa3, 0x80, 0x18, 0x53, 0x80, 0x67, 0xff,
		0x15, 0x49, 0xa9, 0x66, 0x87, 0x2a, 0xd7, 0xb1, 0x83, 0x0f, 0x42, 0x40,
		0x1c, 0x53, 0xbb, 0x6b, 0xaa}
	for i, position := range []uint32{100, uint32(flow.MiB), uint32(3 * flow.MiB)} {
		cue := []byte{0xbb, 0x8c, 0xb3, 0x82, 0, 0, 0xb7, 0x86, 0xf1, 0x84, 0, 0, 0, 0}
		binary.BigEndian.PutUint16(cue[4:6], uint16(i*1000))
		binary.BigEndian.PutUint32(cue[10:14], position)
		header = append(header, cue...)
	}
	payload := make([]byte, pieceBlockSize)
	copy(payload, header)
	if _, err := s.callbackWrite(302, 0, 0, payload); err != nil {
		t.Fatal(err)
	}
	c.MarkComplete(0)
	c.SetFlowMediaEstimate("phone", 1, flow.Estimate{BytesPerSecond: float64(flow.MiB) / 4, Confidence: "high"})
	c.flowMu.Lock()
	g := c.flowGroups["phone"]
	g.anchor = 128
	before := g.pieces
	g.smoother.Reset()
	c.flowMu.Unlock()
	c.requestBurstIndex(FileInfo{Index: 1, Path: "owned.mkv", Length: 64 * flow.MiB})
	c.indexWorkers.Wait()
	c.flowMu.Lock()
	c.refreshFlowWindowLocked(time.Now())
	after, source := g.pieces, g.burstSource
	c.flowMu.Unlock()
	if source != "matroska-cues-coarse" || after <= before {
		t.Fatal("resident index did not reach adaptive demand", source, before, after)
	}
	if len(c.pieces) != 1 || c.ActiveReaders() != 0 {
		t.Fatal("successful inspection fetched media or created readers")
	}
}

func TestResidentIndexReaderNeverFetchesMissingOrUnverifiedBytes(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.ContainerBurstHints = true
	settings.StoreBTsets(&settings.BTSets{CacheSize: 8 << 20, Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(301, mkHash(0x31), 3, pieceBlockSize)
	s.callbackSize(301, 3*pieceBlockSize)
	c := s.lookup(301)
	defer s.callbackClose(301)
	file := FileInfo{Index: 1, Path: "owned.mp4", Offset: pieceBlockSize / 2, Length: 2 * pieceBlockSize}
	reader := residentIndexReader{c, file}
	data := make([]byte, pieceBlockSize)
	if n, err := reader.ReadAt(data, 0); n != 0 || err != io.EOF {
		t.Fatal("missing bytes did not immediately fail")
	}
	if len(c.pieces) != 0 || c.ActiveReaders() != 0 {
		t.Fatal("index read created demand")
	}
	payload := bytes.Repeat([]byte{0x3c}, pieceBlockSize)
	if _, err := s.callbackWrite(301, 0, 0, payload); err != nil {
		t.Fatal(err)
	}
	if n, err := reader.ReadAt(data, 0); n != 0 || err != io.EOF {
		t.Fatal("unverified bytes exposed to index parser")
	}
	// The responsive HTTP gate is intentionally different: forcing verification
	// must happen inside the locked copy, not via a separate, racy Have call.
	if n, err := c.readStreamPiece(0, 0, data); n != len(data) || err != nil {
		t.Fatal("responsive HTTP bytes were disabled", n, err)
	}
	if n, err := c.readResidentPiece(0, 0, data, true); n != 0 || err != io.EOF {
		t.Fatal("locked index copy bypassed verification", n, err)
	}
	c.MarkComplete(0)
	if n, err := reader.ReadAt(data, 0); n != pieceBlockSize/2 || err != io.EOF {
		t.Fatalf("hole crossed: %d %v", n, err)
	}
	if len(c.pieces) != 1 || c.ActiveReaders() != 0 {
		t.Fatal("missing piece was fetched or allocated")
	}
	c.requestBurstIndex(file)
	c.indexWorkers.Wait()
	if len(c.pieces) != 1 || c.ActiveReaders() != 0 {
		t.Fatal("asynchronous parse created download demand")
	}
	c.stopBurstIndexes()
	c.requestBurstIndex(file)
	if c.indexBusy || len(c.indexes) != 0 {
		t.Fatal("parser restarted after close")
	}
}
