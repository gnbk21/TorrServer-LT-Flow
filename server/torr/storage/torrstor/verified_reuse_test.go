package torrstor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"server/flow"
	"server/lt"
	"server/settings"
	"testing"
)

func TestReuseIndexExcludesPrivateUnknownAndIncompatibleFiles(t *testing.T) {
	s := NewStorage()
	root := sha256.Sum256(bytes.Repeat([]byte{1}, 10001))
	makeMeta := func(hash flow.InfoHash, offset int64) *lt.ParsedTorrent {
		return &lt.ParsedTorrent{InfoHash: hash.HexString(), InfoHashV2: hex.EncodeToString(root[:]), PieceLength: 32768, TotalSize: offset + 10001, V2Files: []flow.V2File{{Offset: offset, Size: 10001, Root: hex.EncodeToString(root[:])}}}
	}
	s.callbackOpen(1, mkHash(0x81), 1, 32768)
	source := s.lookup(1)
	s.callbackOpen(2, mkHash(0x82), 2, 32768)
	target := s.lookup(2)
	if _, err := source.writePiece(0, 0, bytes.Repeat([]byte{1}, 10001)); err != nil {
		t.Fatal(err)
	}
	source.SignalPieceComplete(0)
	a, b := makeMeta(source.InfoHash, 0), makeMeta(target.InfoHash, 32768)
	if !s.SetReuseMetadata(source, a) || !s.SetReuseMetadata(target, b) {
		t.Fatal("valid root index failed")
	}
	from, _, _, local := s.reuseCandidate(target, 1)
	if from != source || local != 0 {
		t.Fatal("native offset mapping failed")
	}
	a.Private = true
	if s.SetReuseMetadata(source, a) {
		t.Fatal("private source indexed")
	}
	if from, _, _, _ := s.reuseCandidate(target, 1); from != nil {
		t.Fatal("private source remained reusable")
	}
	a.Private = false
	if !s.SetReuseMetadata(source, a) {
		t.Fatal("public source not restored")
	}
	b.Private = true
	if s.SetReuseMetadata(target, b) {
		t.Fatal("private destination indexed")
	}
	if from, _, _, _ := s.reuseCandidate(target, 1); from != nil {
		t.Fatal("private destination remained reusable")
	}
	b.Private = false
	b.InfoHash = mkHash(0x83).HexString()
	if s.SetReuseMetadata(target, b) {
		t.Fatal("metadata was attached to a different native owner")
	}
	if s.SetReuseMetadata(target, nil) {
		t.Fatal("unknown metadata indexed")
	}
	source.clearReuse()
	if len(s.reuseEntries) != 0 || s.reuseLayerBytes != 0 {
		t.Fatal("closed index retained metadata")
	}
}

func TestReuseYieldsToPressureWaitsDisconnectAndDisablement(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	f := settings.DefaultFlowSettings()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 1 << 20, Flow: f})
	s := NewStorage()
	s.callbackOpen(1, mkHash(0x84), 2, 32768)
	c := s.lookup(1)
	s.callbackOpen(2, mkHash(0x85), 2, 32768)
	other := s.lookup(2)
	if !c.reuseAllowed(32768) {
		t.Fatal("idle compatible cache was rejected")
	}
	other.flowWaiting.Store(1)
	if c.reuseAllowed(32768) {
		t.Fatal("reuse competed with another waiting reader")
	}
	other.flowWaiting.Store(0)
	other.backgroundLimited.Store(true)
	if c.reuseAllowed(32768) {
		t.Fatal("reuse ignored pressure")
	}
	other.backgroundLimited.Store(false)
	c.flowReconnect.Store(true)
	if c.reuseAllowed(32768) {
		t.Fatal("reuse ignored reconnect")
	}
	c.flowReconnect.Store(false)
	c.resourceBudget.Store(1024)
	if c.reuseAllowed(32768) {
		t.Fatal("reuse expanded the cache budget")
	}
	c.resourceBudget.Store(0)
	disabled := *f
	disabled.Enabled = false
	settings.StoreBTsets(&settings.BTSets{CacheSize: 1 << 20, Flow: &disabled})
	if c.reuseAllowed(32768) {
		t.Fatal("disabled Flow admitted reuse")
	}
}

func TestReuseDescriptorBoundAndCapacityRecovery(t *testing.T) {
	s := NewStorage()
	s.callbackOpen(1, mkHash(0x86), reuseDescriptorLimit, 32768)
	s.callbackOpen(2, mkHash(0x87), 1, 32768)
	a, b := s.lookup(1), s.lookup(2)
	root := sha256.Sum256([]byte{7})
	meta := &lt.ParsedTorrent{InfoHash: a.InfoHash.HexString(), InfoHashV2: hex.EncodeToString(root[:]), PieceLength: 32768, TotalSize: reuseDescriptorLimit * 32768}
	for i := 0; i < reuseDescriptorLimit; i++ {
		meta.V2Files = append(meta.V2Files, flow.V2File{Offset: int64(i) * 32768, Size: 1, Root: hex.EncodeToString(root[:])})
	}
	if !s.SetReuseMetadata(a, meta) || len(s.reuseEntries) != reuseDescriptorLimit {
		t.Fatal("bounded descriptor index rejected its boundary")
	}
	other := &lt.ParsedTorrent{InfoHash: b.InfoHash.HexString(), InfoHashV2: meta.InfoHashV2, PieceLength: 32768, TotalSize: 1, V2Files: []flow.V2File{{Size: 1, Root: hex.EncodeToString(root[:])}}}
	if s.SetReuseMetadata(b, other) || len(s.reuseEntries) > reuseDescriptorLimit {
		t.Fatal("descriptor bound exceeded")
	}
	a.clearReuse()
	if !s.SetReuseMetadata(b, other) || len(s.reuseEntries) != 1 || s.reuseLayerBytes != 32 {
		t.Fatal("released index capacity did not recover")
	}
}
