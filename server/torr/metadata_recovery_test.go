package torr

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"server/flow"
	"server/lt"
	"server/settings"
)

func TestMetadataSourcesBoundedAndRequireOptIn(t *testing.T) {
	uri := validMagnet + "&xs=" + url.QueryEscape("https://source.example/file.torrent") +
		"&xs=" + url.QueryEscape("https://source.example/file.torrent") +
		"&as=" + url.QueryEscape("https://second.example/file.torrent") +
		"&as=" + url.QueryEscape("https://third.example/file.torrent")
	spec, err := ParseMagnetURI(uri)
	if err != nil || len(spec.MetadataSources) != 2 {
		t.Fatal("lost bounded supplied URLs", spec, err)
	}
	f := settings.DefaultFlowSettings()
	if sources := metadataSources(spec, f); len(sources) != 2 || sources[0].kind != "SUPPLIED" {
		t.Fatal("unexpected default public disclosure", sources)
	}
	f.PublicMetadataCache = true
	if sources := metadataSources(spec, f); len(sources) != 4 || sources[2].kind != "PUBLIC_CACHE" {
		t.Fatal("public opt-in not honored", sources)
	} else if !strings.HasPrefix(sources[3].url, "https://itorrents.net/torrent/") {
		t.Fatal("cache URL must use the HTTPS host without its HTTP downgrade redirect")
	}
	spec.InfoHash = NewHashFromHex(strings.Repeat("ab", 32))
	if sources := metadataSources(spec, f); len(sources) != 2 {
		t.Fatal("looked up a truncated v2 hash", sources)
	}
	spec.InfoBytes = minimalTorrent()
	if sources := metadataSources(spec, f); len(sources) != 0 {
		t.Fatal("queried caches for known/private metadata")
	}
	spec.InfoBytes = nil
	f.MetadataFallback = false
	if sources := metadataSources(spec, f); len(sources) != 0 {
		t.Fatal("disabled fallback still queried sources")
	}
	for _, raw := range []string{"http://example.com/a", "file:///secret", "https://user:pass@example.com/a", "https://127.0.0.1/a", "https://[::1]/a", "https://10.0.0.1/a", "https://example.com/a#fragment", strings.Repeat("a", 8193)} {
		if validMetadataURL(raw) {
			t.Fatal("unsafe automatic source accepted")
		}
	}
	stored := torrSpec(settingsSpec(spec))
	stored.MetadataSources[0] = "changed"
	if spec.MetadataSources[0] == "changed" {
		t.Fatal("persistence shares mutable source slice")
	}
}

func TestMetadataHTTPBoundsAndCancellation(t *testing.T) {
	payload := minimalTorrent()
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/valid":
			_, _ = w.Write(payload)
		case "/oversized":
			w.Header().Set("Content-Length", "16777217")
		case "/cancel":
			<-r.Context().Done()
		case "/body-timeout":
			w.WriteHeader(http.StatusOK)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/empty":
		case "/compressed-oversized":
			w.Header().Set("Content-Encoding", "gzip")
			compressed := gzip.NewWriter(w)
			_, _ = io.CopyN(compressed, strings.NewReader(strings.Repeat("x", metadataSourceBytes+1)), metadataSourceBytes+1)
			_ = compressed.Close()
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	data, err := readMetadataHTTP(context.Background(), srv.Client(), srv.URL+"/valid")
	if err != nil || !bytes.Equal(data, payload) {
		t.Fatal("valid HTTPS body changed", err)
	}
	for _, path := range []string{"/oversized", "/missing", "/empty", "/compressed-oversized"} {
		if _, err := readMetadataHTTP(context.Background(), srv.Client(), srv.URL+path); err == nil {
			t.Fatal("accepted invalid source response", path)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	if _, err := readMetadataHTTP(ctx, srv.Client(), srv.URL+"/cancel"); err == nil || time.Since(started) > time.Second {
		t.Fatal("cancelled download did not stop promptly")
	}
	bodyCtx, bodyCancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer bodyCancel()
	if _, err := readMetadataHTTP(bodyCtx, srv.Client(), srv.URL+"/body-timeout"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("a timed out body was mislabeled as invalid torrent data", err)
	}
}

type metadataRoundTripper func(*http.Request) (*http.Response, error)

func (f metadataRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type metadataBodyReader func([]byte) (int, error)

func (f metadataBodyReader) Read(p []byte) (int, error) { return f(p) }

func TestMetadataCancellationWinsOverEmptyEOF(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		name := "cancelled"
		if deadline {
			name = "deadline"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				client := &http.Client{Transport: metadataRoundTripper(func(req *http.Request) (*http.Response, error) {
					body := metadataBodyReader(func([]byte) (int, error) {
						if deadline {
							time.Sleep(time.Second)
							synctest.Wait()
						} else {
							cancel()
						}
						// Models a transport reporting EOF when request cancellation
						// closes an empty response body. No real network/timer race.
						return 0, io.EOF
					})
					return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(body), Request: req}, nil
				})}
				_, err := readMetadataHTTP(ctx, client, "https://source.example/metadata")
				want := context.Canceled
				if deadline {
					want = context.DeadlineExceeded
				}
				if !errors.Is(err, want) {
					t.Fatalf("cancellation mislabeled: want %v, got %v", want, err)
				}
			})
		})
	}
}

func TestMetadataRedirectPrivacyAndHTTPSPolicy(t *testing.T) {
	for _, destination := range []string{"https://target.example/a", "http://target.example/a",
		"https://127.0.0.1/a", "https://user:secret@target.example/a", "https://target.example/a#secret"} {
		t.Run(destination, func(t *testing.T) {
			calls := 0
			client := metadataHTTPClient(metadataRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Header.Get("Referer") != "" {
					t.Fatal("source query leaked in redirect Referer")
				}
				response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("metadata")), Request: r}
				if calls == 1 {
					response.StatusCode = http.StatusFound
					response.Header.Set("Location", destination)
				}
				return response, nil
			}))
			_, err := readMetadataHTTP(context.Background(), client, "https://source.example/private?token=secret")
			if destination == "https://target.example/a" {
				if err != nil || calls != 2 {
					t.Fatal("valid HTTPS redirect failed", err, calls)
				}
			} else {
				var refused metadataSourceError
				if !errors.As(err, &refused) || refused.code != "REDIRECT_REFUSED" || calls != 1 {
					t.Fatal("unsafe redirect followed or unclassified", err, calls)
				}
			}
		})
	}
	calls := 0
	client := metadataHTTPClient(metadataRoundTripper(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"https://source.example/loop"}}, Body: io.NopCloser(strings.NewReader("")), Request: r}, nil
	}))
	if _, err := readMetadataHTTP(context.Background(), client, "https://source.example/loop"); err == nil || calls != 3 {
		t.Fatal("redirect loop exceeded bound", err, calls)
	}
}

func TestMetadataFallbackUsesVerifiedSourceWithoutReplacingTorrent(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	settings.StoreBTsets(&settings.BTSets{Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	payload := minimalTorrent()
	parsed, err := lt.ParseTorrentBytes(payload)
	if err != nil {
		t.Fatal(err)
	}
	s, err := lt.NewSession(lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	h, err := s.AddTorrent(lt.AddTorrentParams{Link: "magnet:?xt=urn:btih:" + parsed.InfoHash, SavePath: t.TempDir(), Paused: true})
	if err != nil {
		t.Fatal(err)
	}
	torrent := &Torrent{TorrentSpec: &TorrentSpec{InfoHash: NewHashFromHex(parsed.InfoHash)}, gotInfoCh: make(chan struct{}), closeCh: make(chan struct{}), flowAddedAt: time.Now()}
	torrent.lh.Store(h)
	calls := 0
	torrent.recoverMetadata(context.Background(), []metadataSource{{"https://bad.example/a", "SUPPLIED"}, {"https://good.example/a", "SUPPLIED"}}, func(ctx context.Context, raw string) ([]byte, error) {
		calls++
		if calls == 1 {
			return bytes.ReplaceAll(payload, []byte("test"), []byte("evil")), nil
		}
		return payload, nil
	})
	if calls != 2 || torrent.LTHandle() != h || len(torrent.InfoBytes) == 0 {
		t.Fatal("fallback did not retain/populate original torrent")
	}
	select {
	case <-torrent.gotInfoCh:
	default:
		t.Fatal("Lampa waiters not released")
	}
	status := torrent.FlowStartup()
	if status.MetadataRecovery == nil || status.MetadataRecovery.State != "METADATA_RECOVERED" || status.MetadataRecovery.Attempts != 2 {
		t.Fatal("missing recovery evidence", status)
	}
	status.MetadataRecovery.State = "changed"
	if torrent.FlowStartup().MetadataRecovery.State == "changed" {
		t.Fatal("status exposes mutable recovery snapshot")
	}
	data, _ := json.Marshal(torrent.timeline.Snapshot())
	if bytes.Contains(data, []byte("example")) || bytes.Contains(data, []byte(parsed.InfoHash)) {
		t.Fatal("metadata source identity leaked to incident timeline")
	}
	torrent.metadataRecoveryState(MetadataRecovery{State: "METADATA_MISS"})
	if torrent.FlowStartup().MetadataRecovery.State != "METADATA_RECOVERED" {
		t.Fatal("late failure replaced completed metadata")
	}
	torrent.recoverMetadata(context.Background(), []metadataSource{{"https://unused.example/a", "SUPPLIED"}}, func(context.Context, string) ([]byte, error) {
		t.Fatal("retried after metadata was already ready")
		return nil, io.EOF
	})
}

func TestMetadataRecoveryRetainsSafeFailureClassification(t *testing.T) {
	dir := t.TempDir()
	history, err := flow.NewHistory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer history.Close(time.Second)
	bt := &BTServer{}
	bt.history.Store(history)
	torrent := &Torrent{bt: bt, flowAddedAt: time.Now()}
	torrent.metadataRecoveryState(MetadataRecovery{State: "METADATA_MISS", Source: "PUBLIC_CACHE", Attempts: 2, Code: "REDIRECT_REFUSED"})
	if !history.Close(time.Second) {
		t.Fatal("recorder did not flush")
	}
	data, err := os.ReadFile(filepath.Join(dir, "flow-history.jsonl"))
	if err != nil || !bytes.Contains(data, []byte(`"code":9`)) || !bytes.Contains(data, []byte(`"operation":2`)) || !bytes.Contains(data, []byte(`"count":2`)) {
		t.Fatal("recorder lost safe source failure classification", err, string(data))
	}
}

func TestMetadataRecoveryOwnerCancelsOnCloseAndCoalescesWaiters(t *testing.T) {
	old := settings.BTsets()
	settings.StoreBTsets(&settings.BTSets{Flow: settings.DefaultFlowSettings()})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	torrent := &Torrent{TorrentSpec: &TorrentSpec{InfoHash: NewHashFromHex(validHashHex), MetadataSources: []string{"https://source.example/a"}},
		gotInfoCh: make(chan struct{}), closeCh: make(chan struct{}), flowAddedAt: time.Now()}
	s, err := lt.NewSession(lt.SessionConfig{"enable_dht": false, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	h, err := s.AddTorrent(lt.AddTorrentParams{Link: validMagnet, SavePath: t.TempDir(), Paused: true})
	if err != nil {
		t.Fatal(err)
	}
	torrent.lh.Store(h)
	started, stopped := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	fetch := func(ctx context.Context, raw string) ([]byte, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-ctx.Done()
		close(stopped)
		return nil, ctx.Err()
	}
	for i := 0; i < 8; i++ {
		torrent.startMetadataRecoveryAfter(time.Millisecond, fetch)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("automatic timer never triggered fallback")
	}
	torrent.Close()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("removed torrent left a source request running")
	}
	if calls.Load() != 1 {
		t.Fatal("waiters multiplied recovery work", calls.Load())
	}
}
