package torr

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"server/flow"
	"server/lt"
	"server/netpolicy"
	"server/settings"
	"server/torr/state"
)

const metadataRecoveryDelay = 15 * time.Second
const metadataSourceTimeout = 8 * time.Second
const metadataSourceBytes = 16 << 20

type metadataSource struct{ url, kind string }

type metadataSourceError struct{ code string }

func (e metadataSourceError) Error() string { return e.code }

// MetadataRecovery exposes enums/counts only. Never publish source URLs,
// credentials, torrent identities or upstream error text in diagnostic history.
type MetadataRecovery struct {
	State    string `json:"state"`
	Source   string `json:"source,omitempty"`
	Attempts int    `json:"attempts"`
	Code     string `json:"code,omitempty"`
}

func validMetadataURL(raw string) bool {
	if len(raw) == 0 || len(raw) > 8192 {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && netpolicy.ValidateDestination(u, false) == nil
}

func magnetMetadataSources(uri string) []string {
	u, err := url.Parse(uri)
	if err != nil {
		return nil
	}
	var sources []string
	seen := map[string]bool{}
	for _, key := range []string{"xs", "as"} {
		for _, raw := range u.Query()[key] {
			if validMetadataURL(raw) && !seen[raw] {
				sources = append(sources, raw)
				seen[raw] = true
				if len(sources) == 2 {
					return sources
				}
			}
		}
	}
	return sources
}

func metadataSources(spec *TorrentSpec, f *settings.FlowSettings) []metadataSource {
	if spec == nil || f == nil || !f.Enabled || !f.MetadataFallback || len(spec.InfoBytes) > 0 {
		return nil
	}
	var out []metadataSource
	seen := map[string]bool{}
	for _, raw := range spec.MetadataSources {
		if validMetadataURL(raw) && !seen[raw] {
			out = append(out, metadataSource{raw, "SUPPLIED"})
			seen[raw] = true
			if len(out) == 2 {
				break
			}
		}
	}
	// These services index v1 hashes only. Never truncate a v2 hash or guess a
	// replacement release. Public lookups require the owner's explicit opt-in.
	if f.PublicMetadataCache && !spec.InfoHash.IsZero() && len(spec.InfoHash.HexString()) == 40 {
		hash := strings.ToUpper(spec.InfoHash.HexString())
		for _, raw := range []string{"https://torrage.info/torrent.php?h=" + hash,
			"https://itorrents.net/torrent/" + hash + ".torrent"} {
			if !seen[raw] {
				out = append(out, metadataSource{raw, "PUBLIC_CACHE"})
			}
		}
	}
	return out
}

// The existing FetchGroup bounds/coalesces concurrent downloads and includes
// network policy in its key. Completed results are not globally cached.
var metadataFetches flow.FetchGroup[[]byte]

func metadataFetchKey(raw string) string {
	policy, _ := json.Marshal(netpolicy.Snapshot())
	digest := sha256.Sum256(append(append([]byte(raw), 0), policy...))
	return hex.EncodeToString(digest[:])
}

func fetchMetadataSource(ctx context.Context, raw string) ([]byte, error) {
	if !validMetadataURL(raw) {
		return nil, errors.New("metadata destination refused")
	}
	key := metadataFetchKey(raw)
	return metadataFetches.Do(ctx, key, 4, func(shared context.Context) ([]byte, error) {
		bounded, cancel := context.WithTimeout(shared, metadataSourceTimeout)
		defer cancel()
		transport := netpolicy.DestinationTransport(false)
		defer transport.CloseIdleConnections()
		return readMetadataHTTP(bounded, metadataHTTPClient(transport), raw)
	})
}

func metadataHTTPClient(transport http.RoundTripper) *http.Client {
	return &http.Client{Transport: transport, Timeout: metadataSourceTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			req.Header.Del("Referer") // supplied source queries can contain private tokens
			if len(via) >= 3 || !validMetadataURL(req.URL.String()) {
				return metadataSourceError{"REDIRECT_REFUSED"}
			}
			return nil
		}}
}

func readMetadataHTTP(ctx context.Context, client *http.Client, raw string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/x-bittorrent, application/octet-stream")
	req.Header.Set("User-Agent", "TorrServer-Flow metadata recovery")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > metadataSourceBytes {
		return nil, metadataSourceError{"HTTP_STATUS_OR_SIZE"}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, metadataSourceBytes+1))
	if err != nil || len(data) == 0 || len(data) > metadataSourceBytes {
		return nil, metadataSourceError{"HTTP_BODY_INVALID"}
	}
	return data, nil
}

func (t *Torrent) metadataRecoveryState(value MetadataRecovery) {
	t.mu.Lock()
	select {
	case <-t.gotInfoCh:
		if value.State != "METADATA_RECOVERED" {
			t.mu.Unlock()
			return // a peer won the race; do not replace its success with a miss
		}
	default:
	}
	t.flowStartup.MetadataRecovery = &value
	t.mu.Unlock()
	codes := map[string]int{"SOURCE_NETWORK_ERROR": 1, "HTTP_STATUS_OR_SIZE": 2,
		"HTTP_BODY_INVALID": 3, "INVALID_METADATA": 4, "HASH_MISMATCH": 5,
		"NATIVE_REJECTED": 6, "FETCH_CAPACITY": 7, "HASH_VERIFIED": 8, "REDIRECT_REFUSED": 9}
	operation := 0
	if value.Source == "SUPPLIED" {
		operation = 1
	} else if value.Source == "PUBLIC_CACHE" {
		operation = 2
	}
	t.historyEvent(flow.HistoryEvent{Type: "metadata_recovery", Stage: value.State, Count: uint64(value.Attempts), Code: codes[value.Code], Operation: operation})
}

func (t *Torrent) startMetadataRecovery() {
	t.startMetadataRecoveryAfter(metadataRecoveryDelay, fetchMetadataSource)
}

func (t *Torrent) startMetadataRecoveryAfter(delay time.Duration, fetch func(context.Context, string) ([]byte, error)) {
	t.metadataRecoveryOnce.Do(func() {
		t.mu.Lock()
		spec := *t.TorrentSpec
		spec.MetadataSources = append([]string(nil), spec.MetadataSources...)
		t.mu.Unlock()
		sources := metadataSources(&spec, settings.CurrentFlow())
		if len(sources) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), delay+4*metadataSourceTimeout+time.Second)
		t.mu.Lock()
		if t.Stat == state.TorrentClosed {
			t.mu.Unlock()
			cancel()
			return
		}
		t.metadataRecoveryCancel = cancel
		t.mu.Unlock()
		t.metadataRecoveryState(MetadataRecovery{State: "METADATA_WAIT"})
		go func() {
			defer cancel()
			timer := time.NewTimer(delay)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return
			case <-t.gotInfoCh:
				return
			case <-timer.C:
			}
			t.recoverMetadata(ctx, sources, fetch)
		}()
	})
}

func (t *Torrent) recoverMetadata(ctx context.Context, sources []metadataSource, fetch func(context.Context, string) ([]byte, error)) {
	last := MetadataRecovery{State: "METADATA_MISS", Code: "NO_MATCHING_METADATA"}
	for _, source := range sources {
		f := settings.CurrentFlow()
		if ctx.Err() != nil || !f.Enabled || !f.MetadataFallback || source.kind == "PUBLIC_CACHE" && !f.PublicMetadataCache {
			break
		}
		handle := t.LTHandle()
		if handle == nil {
			return
		}
		if have, _ := handle.HaveMetadata(); have {
			t.signalGotInfo()
			return
		}
		last.Attempts++
		last.Source, last.State, last.Code = source.kind, "METADATA_HTTP", ""
		t.metadataRecoveryState(last)
		data, err := fetch(ctx, source.url)
		last.Code = "SOURCE_NETWORK_ERROR"
		if errors.Is(err, flow.ErrFetchBusy) {
			last.Code = "FETCH_CAPACITY"
		}
		var sourceError metadataSourceError
		if errors.As(err, &sourceError) {
			last.Code = sourceError.code
		}
		if err == nil && ctx.Err() == nil {
			var parsed *lt.ParsedTorrent
			parsed, err = lt.ParseTorrentBytes(data)
			last.Code = "INVALID_METADATA"
			if err == nil && (!parsed.HasMetadata || t.Hash().HexString() != parsed.InfoHashV1 && t.Hash().HexString() != parsed.InfoHashV2) {
				err = errors.New("metadata hash mismatch")
				last.Code = "HASH_MISMATCH"
			}
			installed := false
			if err == nil {
				// Native code checks all known hashes again. Keep the handle and
				// its authorized tracker tiers; ignore outer trackers/web seeds.
				installed, err = handle.SetMetadata(data)
				last.Code = "NATIVE_REJECTED"
			}
			if err == nil {
				t.signalGotInfo()
				if have, _ := handle.HaveMetadata(); have && installed {
					last.State, last.Code = "METADATA_RECOVERED", "HASH_VERIFIED"
					t.metadataRecoveryState(last)
					return
				}
				return // a peer completed the same metadata concurrently
			}
		}
		last.State = "METADATA_MISS"
		t.metadataRecoveryState(last)
	}
}
