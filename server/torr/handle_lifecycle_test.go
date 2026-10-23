package torr

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"runtime"
	"server/lt"
	"server/settings"
	"server/torr/storage/torrstor"
	"sync"
	"testing"
	"time"
)

func TestWaitInfoUnblocksOnConcurrentNativeRemoval(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{TorrentDisconnectTimeout: 30, Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	session, err := lt.NewSession(lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	bt := &BTServer{session: session, torrents: map[Hash]*Torrent{}}
	for iteration := 0; iteration < 16; iteration++ {
		hash := fmt.Sprintf("%040x", iteration+1)
		handle, err := session.AddTorrent(lt.AddTorrentParams{Link: "magnet:?xt=urn:btih:" + hash, SavePath: t.TempDir(), Paused: true})
		if err != nil {
			t.Fatal(err)
		}
		torrent := &Torrent{bt: bt, TorrentSpec: &TorrentSpec{InfoHash: NewHashFromHex(hash)}, gotInfoCh: make(chan struct{}), closeCh: make(chan struct{})}
		torrent.lh.Store(handle)
		results := make(chan bool, 8)
		start := make(chan struct{})
		for worker := 0; worker < cap(results); worker++ {
			go func() { <-start; results <- torrent.WaitInfo() }()
		}
		close(start)
		torrent.Close()
		for worker := 0; worker < cap(results); worker++ {
			select {
			case ready := <-results:
				if ready {
					t.Fatal("removed metadata-less torrent reported ready")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("metadata waiter outlived removal")
			}
		}
		if torrent.WaitInfo() {
			t.Fatal("closed torrent reported metadata ready")
		}
	}
	var absent *Torrent
	if absent.WaitInfo() {
		t.Fatal("nil torrent reported ready")
	}
}

// Metadata/preload callers can retain a snapshot while cleanup detaches it.
// Exercise the actual Close/LTHandle paths under the race detector.
func TestCloseDetachesHandleWhileCallersSnapshot(t *testing.T) {
	for iteration := 0; iteration < 32; iteration++ {
		torrent := &Torrent{closeCh: make(chan struct{})}
		handle := &lt.Torrent{}
		torrent.lh.Store(handle)
		var callers sync.WaitGroup
		started := make(chan struct{}, 4)
		for worker := 0; worker < 4; worker++ {
			callers.Go(func() {
				started <- struct{}{}
				for i := 0; i < 1000; i++ {
					if got := torrent.LTHandle(); got != nil && got != handle {
						t.Error("published a different native identity")
					}
				}
			})
		}
		for worker := 0; worker < 4; worker++ {
			<-started
		}
		if !torrent.Close() || !torrent.Close() {
			t.Fatal("close must remain idempotent")
		}
		callers.Wait()
		if torrent.LTHandle() != nil {
			t.Fatal("closed torrent retained its published handle")
		}
	}
}

// Exercise real native identity and cache lifetime, rather than a zero-valued
// handle. A retirement racing revival must not invalidate the fresh instance.
func TestNativeRetirementCannotRemoveRevivedHandle(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.PeerResumeHints = false
	settings.StoreBTsets(&settings.BTSets{CacheSize: 4 << 20, DisableDHT: true, Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	storage := torrstor.Global()
	if err := storage.Install(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = storage.Uninstall() })
	session, err := lt.NewSession(lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	digest := sha1.Sum(bytes.Repeat([]byte{0x37}, 16384))
	payload := append([]byte("d4:infod6:lengthi16384e4:name4:test12:piece lengthi16384e6:pieces20:"), digest[:]...)
	payload = append(payload, []byte("ee")...)
	metadata, err := lt.ParseTorrentBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	bt := &BTServer{session: session, torrents: map[Hash]*Torrent{}}
	spec := &TorrentSpec{InfoHash: NewHashFromHex(metadata.InfoHash), InfoBytes: payload}
	for iteration := 0; iteration < 128; iteration++ {
		retiring, err := NewTorrent(spec, bt)
		if err != nil {
			t.Fatal(err)
		}
		done := make(chan struct{})
		go func() { retiring.Close(); close(done) }()
		deadline := time.Now().Add(time.Second)
		for retiring.LTHandle() != nil && time.Now().Before(deadline) {
			runtime.Gosched()
		}
		if retiring.LTHandle() != nil {
			t.Fatal("retirement did not detach the handle")
		}
		fresh, err := NewTorrent(spec, bt)
		<-done
		if err != nil || fresh == nil || fresh == retiring {
			t.Fatal("revival failed", err)
		}
		if handle := fresh.LTHandle(); handle == nil {
			t.Fatal("revival lost its handle")
		} else if st, err := handle.Status(); err != nil || st == nil || !st.HasMetadata {
			t.Fatal("retirement removed the revived native identity", err)
		}
		if storage.CacheByHash(fresh.StorageHash()) == nil {
			t.Fatal("revival lost its cache")
		}
		bt.RemoveTorrent(fresh.Hash())
	}
}
