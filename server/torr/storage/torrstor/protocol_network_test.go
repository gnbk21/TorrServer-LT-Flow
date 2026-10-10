package torrstor

import (
	"bytes"
	"context"
	"crypto/sha1"
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

// Different short files exercise native pad-file indices, full piece layers,
// SHA-256 verification, and a short final block. No resume/have override is used.
func networkProtocolFixture(hybrid bool) ([]byte, map[string][]byte) {
	a := bytes.Repeat([]byte{0x37}, 17001)
	b := bytes.Repeat([]byte{0x92}, 33017)
	pair := func(x, y [32]byte) [32]byte {
		var data [64]byte
		copy(data[:32], x[:])
		copy(data[32:], y[:])
		return sha256.Sum256(data[:])
	}
	rootA := pair(sha256.Sum256(a[:16384]), sha256.Sum256(a[16384:]))
	firstB := pair(sha256.Sum256(b[:16384]), sha256.Sum256(b[16384:32768]))
	lastB := pair(sha256.Sum256(b[32768:]), [32]byte{})
	rootB := pair(firstB, lastB)
	info := fmt.Sprintf("d9:file treed5:a.mkvd0:d6:lengthi17001e11:pieces root32:%see5:b.mkvd0:d6:lengthi33017e11:pieces root32:%seee", string(rootA[:]), string(rootB[:]))
	if hybrid {
		// Canonical hybrid layout includes virtual alignment padding between files.
		info += "5:filesld6:lengthi17001e4:pathl5:a.mkveed4:attr1:p6:lengthi15767e4:pathl4:.pad5:15767eed6:lengthi33017e4:pathl5:b.mkveee"
	}
	info += "12:meta versioni2e4:name6:series12:piece lengthi32768e"
	if hybrid {
		first := append(append([]byte(nil), a...), make([]byte, 15767)...)
		hashes := [][20]byte{sha1.Sum(first), sha1.Sum(b[:32768]), sha1.Sum(b[32768:])}
		info += "6:pieces60:"
		for _, h := range hashes {
			info += string(h[:])
		}
	}
	info += "e"
	payload := []byte("d4:info" + info + "12:piece layersd32:" + string(rootB[:]) + "64:" + string(firstB[:]) + string(lastB[:]) + "ee")
	return payload, map[string][]byte{"a.mkv": a, "b.mkv": b}
}

func TestNativeV2AndHybridNetworkTransferIntoVerifiedStreamingCache(t *testing.T) {
	for _, hybrid := range []bool{false, true} {
		t.Run(fmt.Sprintf("hybrid=%t", hybrid), func(t *testing.T) {
			payload, content := networkProtocolFixture(hybrid)
			metadata, err := lt.ParseTorrentBytes(payload)
			if err != nil {
				t.Fatal("fixture metadata", err)
			}
			if len(metadata.V2Files) != 2 || !metadata.CompletePieceLayers() || metadata.NumPieces != 3 {
				t.Fatal("incomplete native geometry", metadata.NumPieces)
			}
			old := settings.BTsets()
			settings.StoreBTsets(&settings.BTSets{CacheSize: 4 << 20, Flow: settings.DefaultFlowSettings()})
			t.Cleanup(func() { settings.StoreBTsets(old) })
			storage := NewStorage()
			if err := storage.Uninstall(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = storage.Uninstall() })
			cfg := lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0", "alert_mask": 0x7fffffff, "enable_outgoing_utp": false, "enable_incoming_utp": false}
			seed, err := lt.NewSession(cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = seed.Close() })
			seedDir := t.TempDir()
			if err := os.Mkdir(filepath.Join(seedDir, "series"), 0700); err != nil {
				t.Fatal(err)
			}
			for name, data := range content {
				if err := os.WriteFile(filepath.Join(seedDir, "series", name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			seedTorrent, err := seed.AddTorrent(lt.AddTorrentParams{InfoBytes: payload, SavePath: seedDir})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = seedTorrent.Remove(false) })
			seedPort := 0
			seedReady := false
			deadline := time.Now().Add(15 * time.Second)
			for time.Now().Before(deadline) {
				alerts, err := seed.PopAlerts()
				if err != nil {
					t.Fatal(err)
				}
				for _, alert := range alerts {
					if alert.Type == "listen_succeeded" && alert.Transport == "tcp" && !alert.IPv6 {
						seedPort = alert.Port
					}
					if alert.Type == "torrent_error" || alert.Type == "file_error" {
						t.Fatal("seed storage error", alert.ErrorCode)
					}
				}
				seedReady = true
				for piece := 0; piece < metadata.NumPieces; piece++ {
					seedReady = seedReady && seedTorrent.HasPiece(piece)
				}
				if seedReady && seedPort > 0 {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !seedReady || seedPort == 0 {
				t.Fatal("native seed did not hash its real files or bind loopback")
			}
			if err := storage.Install(); err != nil {
				t.Fatal(err)
			}
			hash := flow.ParseInfoHash(metadata.InfoHash)
			storage.RequireVerifiedReads(hash)
			receiver, err := lt.NewSession(cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = receiver.Close() })
			h, err := receiver.AddTorrent(lt.AddTorrentParams{InfoBytes: payload, SavePath: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = h.Remove(false) })
			if err := h.RestorePeerHints([]flow.PeerHint{{IP: "127.0.0.1", Port: seedPort}}); err != nil {
				t.Fatal(err)
			}
			var cache *Cache
			complete := false
			deadline = time.Now().Add(20 * time.Second)
			for time.Now().Before(deadline) {
				alerts, err := receiver.PopAlerts()
				if err != nil {
					t.Fatal(err)
				}
				cache = storage.CacheByHash(hash)
				for _, alert := range alerts {
					if alert.Type == "torrent_error" || alert.Type == "file_error" || alert.Type == "hash_failed" {
						t.Fatal("native transfer/hash failure", alert.Type, alert.ErrorCode)
					}
					if alert.Type == "piece_finished" && cache != nil {
						cache.SignalPieceComplete(alert.Piece)
					}
				}
				complete = cache != nil
				for piece := 0; piece < metadata.NumPieces; piece++ {
					complete = complete && cache != nil && cache.Have(piece)
				}
				if complete {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !complete {
				st, _ := h.Status()
				t.Fatal("native transfer never verified every piece", st.NumPieces, st.NumPeers, st.TotalPayloadDownload)
			}
			st, err := h.Status()
			if err != nil || st.TotalPayloadDownload < int64(len(content["a.mkv"])+len(content["b.mkv"])) {
				t.Fatal("test bypassed real network payload", err, st.TotalPayloadDownload)
			}
			files, err := h.Files()
			if err != nil || len(files) != 2 {
				t.Fatal("padding leaked into file selection", err, len(files))
			}
			for _, file := range files {
				want := content[filepath.Base(file.Path)]
				if want == nil {
					t.Fatal("unexpected selected file")
				}
				r := NewReader(cache, h, FileInfo{Index: file.Index, Path: file.Path, Offset: file.Offset, Length: file.Size})
				got, readErr := io.ReadAll(r)
				_ = r.Close()
				if readErr != nil || !bytes.Equal(got, want) {
					t.Fatal("verified streaming bytes changed", file.Index, readErr, len(got), len(want))
				}
			}
			verifyNativeReuse(t, storage, receiver, cache, metadata, payload, content)
		})
	}
}

func verifyNativeReuse(t *testing.T, storage *Storage, session *lt.Session, source *Cache, metadata *lt.ParsedTorrent, payload []byte, content map[string][]byte) {
	t.Helper()
	if !storage.SetReuseMetadata(source, metadata) {
		t.Fatal("verified source metadata was not indexed")
	}
	targetPayload := bytes.Replace(payload, []byte("4:name6:series"), []byte("4:name6:copy00"), 1)
	targetMetadata, err := lt.ParseTorrentBytes(targetPayload)
	if err != nil {
		t.Fatal(err)
	}
	h, err := session.AddTorrent(lt.AddTorrentParams{InfoBytes: targetPayload, SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = h.Remove(false) })
	_ = h.SetAllPiecesPriority(0)
	targetHash := flow.ParseInfoHash(targetMetadata.InfoHash)
	var target *Cache
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		target = storage.CacheByHash(targetHash)
		st, _ := h.Status()
		if target != nil && st != nil && st.State != "checking_files" && st.State != "checking_resume_data" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if target == nil || !storage.SetReuseMetadata(target, targetMetadata) {
		t.Fatal("distinct target metadata was not indexed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	// Simulate changed backing bytes despite a previously verified source bit.
	// Reuse must rehash rather than trust that old completion flag.
	source.mu.RLock()
	sourcePiece := source.pieces[0]
	source.mu.RUnlock()
	if _, err := sourcePiece.WriteAt([]byte{0xff}, 0); err != nil {
		t.Fatal(err)
	}
	if !target.tryReusePiece(ctx, nil, h, 0) {
		t.Fatal("corruption check was not scheduled")
	}
	for reuseWorker.Load() && ctx.Err() == nil {
		time.Sleep(time.Millisecond)
	}
	if ctx.Err() != nil || target.Have(0) || target.Filled() != 0 {
		t.Fatal("corrupt source bytes reached the target")
	}
	if _, err := sourcePiece.WriteAt(content["a.mkv"][:1], 0); err != nil {
		t.Fatal(err)
	}
	for piece := 0; piece < targetMetadata.NumPieces; piece++ {
		for !target.Have(piece) && ctx.Err() == nil {
			target.tryReusePiece(ctx, nil, h, piece)
			time.Sleep(20 * time.Millisecond)
			alerts, err := session.PopAlerts()
			if err != nil {
				t.Fatal(err)
			}
			for _, alert := range alerts {
				if alert.TorrentHash != targetHash.HexString() {
					continue
				}
				if alert.Type == "hash_failed" || alert.Type == "file_error" || alert.Type == "torrent_error" {
					t.Fatal("reused bytes failed native hashing", alert.Type, alert.ErrorCode)
				}
				if alert.Type == "piece_finished" {
					target.SignalPieceComplete(alert.Piece)
				}
			}
			if !target.Have(piece) && target.readableAt(piece, 0) > 0 {
				t.Fatal("reuse exposed bytes before native verification")
			}
		}
		if !target.Have(piece) {
			t.Fatal("native reuse did not complete", piece, ctx.Err())
		}
	}
	st, err := h.Status()
	if err != nil || st.TotalPayloadDownload != 0 || st.NumPeers != 0 {
		t.Fatal("reuse accidentally downloaded from another swarm", err)
	}
	files, err := h.Files()
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		r := NewReader(target, h, FileInfo{Index: file.Index, Path: file.Path, Offset: file.Offset, Length: file.Size})
		got, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil || !bytes.Equal(got, content[filepath.Base(file.Path)]) {
			t.Fatal("reused stream bytes changed", err)
		}
	}
	storage.reuseMu.Lock()
	before := len(storage.reuseEntries)
	storage.reuseMu.Unlock()
	target.clearReuse()
	storage.reuseMu.Lock()
	after := len(storage.reuseEntries)
	storage.reuseMu.Unlock()
	if after >= before {
		t.Fatal("closed owner retained reuse descriptors")
	}
}
