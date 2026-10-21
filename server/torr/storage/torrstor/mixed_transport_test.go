package torrstor

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"server/flow"
	"server/lt"
	"server/settings"
)

// Two complementary real native seeds require both transports. Each has only
// half the 8 MiB pieces; two device windows request opposite ends concurrently.
func TestEightMiBPiecesFromMixedTCPAndUTPSeeds(t *testing.T) {
	const piece = 8 << 20
	data := make([]byte, 4*piece)
	for n := 0; n < 4; n++ {
		copy(data[n*piece:(n+1)*piece], bytes.Repeat([]byte{byte(0x37 + n)}, piece))
	}
	var hashes []byte
	for off := 0; off < len(data); off += piece {
		sum := sha1.Sum(data[off : off+piece])
		hashes = append(hashes, sum[:]...)
	}
	metadata := append([]byte(fmt.Sprintf("d4:infod6:lengthi%de4:name9:mixed.bin12:piece lengthi%de6:pieces80:", len(data), piece)), hashes...)
	metadata = append(metadata, []byte("ee")...)
	parsed, err := lt.ParseTorrentBytes(metadata)
	if err != nil {
		t.Fatal(err)
	}
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 << 20, Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	storage := NewStorage()
	if err := storage.Uninstall(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = storage.Uninstall() })
	config := func(tcp, utp bool) lt.SessionConfig {
		return lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0", "alert_mask": 0x7fffffff, "enable_incoming_tcp": tcp, "enable_outgoing_tcp": tcp, "enable_incoming_utp": utp, "enable_outgoing_utp": utp, "close_redundant_connections": false, "allow_multiple_connections_per_ip": true}
	}
	var seeds []*lt.Session
	var hints []flow.PeerHint
	for n := 0; n < 2; n++ {
		seed, err := lt.NewSession(config(n == 0, n == 1))
		if err != nil {
			t.Fatal(err)
		}
		seeds = append(seeds, seed)
		t.Cleanup(func() { _ = seed.Close() })
		dir := t.TempDir()
		partial := make([]byte, len(data))
		copy(partial[n*2*piece:(n+1)*2*piece], data[n*2*piece:(n+1)*2*piece])
		if err := os.WriteFile(filepath.Join(dir, "mixed.bin"), partial, 0600); err != nil {
			t.Fatal(err)
		}
		h, err := seed.AddTorrent(lt.AddTorrentParams{InfoBytes: metadata, SavePath: dir})
		if err != nil {
			t.Fatal(err)
		}
		_ = h.SetAllPiecesPriority(0)
		port := 0
		ready := false
		deadline := time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			alerts, err := seed.PopAlerts()
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range alerts {
				if a.Type == "listen_succeeded" && !a.IPv6 && a.Transport == []string{"tcp", "udp"}[n] {
					port = a.Port
				}
			}
			ready = h.HasPiece(n*2) && h.HasPiece(n*2+1)
			if ready && port > 0 {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !ready || port == 0 {
			t.Fatal("partial native seed not ready", n)
		}
		hints = append(hints, flow.PeerHint{IP: "127.0.0.1", Port: port})
	}
	if err := storage.Install(); err != nil {
		t.Fatal(err)
	}
	cfg := config(true, true)
	cfg["download_rate_limit"] = 4 << 20
	receiver, err := lt.NewSession(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = receiver.Close() })
	h, err := receiver.AddTorrent(lt.AddTorrentParams{InfoBytes: metadata, SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	_ = h.SetAllPiecesPriority(0)
	if err := h.RestorePeerHints(hints); err != nil {
		t.Fatal(err)
	}
	var cache *Cache
	deadline := time.Now().Add(10 * time.Second)
	for cache == nil && time.Now().Before(deadline) {
		cache = storage.CacheByHash(flow.ParseInfoHash(parsed.InfoHash))
		time.Sleep(10 * time.Millisecond)
	}
	if cache == nil {
		t.Fatal("receiver cache unavailable")
	}
	results := make(chan error, 2)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	t.Cleanup(cancel)
	for n := 0; n < 2; n++ {
		r := NewReader(cache, h, FileInfo{Index: 1, Path: "mixed.bin", Length: int64(len(data))}, fmt.Sprintf("device-%d", n))
		r.SetContext(ctx)
		go func(n int, r *Reader) {
			defer r.Close()
			off := int64(n * 3 * piece)
			if _, err := r.Seek(off, io.SeekStart); err != nil {
				results <- err
				return
			}
			got := make([]byte, 65536)
			_, err := io.ReadFull(r, got)
			if err == nil && !bytes.Equal(got, data[off:off+int64(len(got))]) {
				err = fmt.Errorf("changed transport bytes")
			}
			results <- err
		}(n, r)
	}
	completed := 0
	seen := map[string]bool{}
	deadline = time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) && (completed < 2 || len(seen) < 2) {
		alerts, _ := receiver.PopAlerts()
		for _, a := range alerts {
			if a.Type == "piece_finished" {
				cache.SignalPieceComplete(a.Piece)
			}
		}
		for _, seed := range seeds {
			alerts, _ := seed.PopAlerts()
			for _, a := range alerts {
				if a.Type == "incoming_connection" {
					seen[a.Transport] = true
				}
			}
		}
		select {
		case err := <-results:
			if err != nil {
				t.Fatal(err)
			}
			completed++
		default:
		}
		time.Sleep(20 * time.Millisecond)
	}
	if completed != 2 || !seen["tcp"] || !seen["utp"] {
		t.Fatal("both real transports and device windows were not observed", completed, seen)
	}
}
