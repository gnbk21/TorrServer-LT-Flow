package torrstor

import (
	"bytes"
	"io"
	"testing"

	"server/settings"
)

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
