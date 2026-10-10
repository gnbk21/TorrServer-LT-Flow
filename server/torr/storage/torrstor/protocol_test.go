package torrstor

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"server/flow"
	"server/lt"
	"server/settings"
	"testing"
	"time"
)

func TestV2NativeResumeReadsShortFileFromFullIdentityDirectory(t *testing.T) {
	content := bytes.Repeat([]byte{0x53}, 17001)
	first := sha256.Sum256(content[:16384])
	last := sha256.Sum256(content[16384:])
	var pair [64]byte
	copy(pair[:32], first[:])
	copy(pair[32:], last[:])
	root := sha256.Sum256(pair[:])
	info := []byte(fmt.Sprintf("d9:file treed11:episode.mkvd0:d6:lengthi17001e11:pieces root32:%seee12:meta versioni2e4:name11:episode.mkv12:piece lengthi32768ee", string(root[:])))
	payload := append([]byte("d4:info"), info...)
	payload = append(payload, 'e')
	metadata, err := lt.ParseTorrentBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	hash := flow.ParseInfoHash(metadata.InfoHash)
	if !hash.IsV2() {
		t.Fatal("v2 metadata lacks complete SHA-256 identity")
	}
	old := settings.BTsets()
	dir := t.TempDir()
	settings.StoreBTsets(&settings.BTSets{UseDisk: true, TorrentsSavePath: dir})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	cacheDir := filepath.Join(dir, hash.HexString())
	if err = os.MkdirAll(cacheDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(cacheDir, "0"), content, 0600); err != nil {
		t.Fatal(err)
	}
	bitmap := ScanMetadataPieces(hash, metadata.PieceLength, metadata.TotalSize, nil, metadata.V2Files)
	if len(bitmap) != 1 || bitmap[0] != 1 {
		t.Fatal("short v2 file was not authenticated", bitmap)
	}
	s := NewStorage()
	s.SetVerifiedResume(hash, bitmap, metadata.TotalSize)
	if err = s.Install(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Uninstall() })
	session, err := lt.NewSession(lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	h, err := session.AddTorrent(lt.AddTorrentParams{InfoBytes: payload, SavePath: dir, Paused: true, HavePieces: bitmap, PieceCount: metadata.NumPieces})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Remove(false) })
	var cache *Cache
	for until := time.Now().Add(3 * time.Second); time.Now().Before(until); {
		cache = s.CacheByHash(hash)
		if cache != nil && cache.Have(0) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if cache == nil || !cache.Have(0) {
		t.Fatal("full native identity/geometry/resume was not installed")
	}
	cache.mu.RLock()
	p := cache.pieces[0]
	cache.mu.RUnlock()
	if p == nil || p.expectedSize() != int64(len(content)) {
		t.Fatal("short file tail changed")
	}
	r := NewReader(cache, h, FileInfo{Index: 0, Path: "episode.mkv", Length: int64(len(content))})
	defer r.Close()
	actual, err := io.ReadAll(r)
	if err != nil || !bytes.Equal(actual, content) {
		t.Fatal("verified v2 streaming bytes changed", err)
	}
}
