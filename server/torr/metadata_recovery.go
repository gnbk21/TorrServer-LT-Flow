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

var metadataHealth flow.SourceHealth

func metadataServiceKey(raw string) string {
	u, _ := url.Parse(raw)
	if u == nil {
		return metadataFetchKey("")
	}
	return metadataFetchKey(u.Scheme + "://" + u.Host)
}

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
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
			return nil, metadataSourceError{"SOURCE_MISSING"}
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			return nil, metadataSourceError{"SOURCE_UNAVAILABLE"}
		}
		return nil, metadataSourceError{"HTTP_STATUS_OR_SIZE"}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, metadataSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > metadataSourceBytes {
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
	codes["SOURCE_MISSING"], codes["SOURCE_UNAVAILABLE"], codes["SOURCE_COOLDOWN"] = 10, 11, 12
	codes["POLICY_CHANGED"], codes["SOURCE_TIMEOUT"] = 13, 14
	codes["SOURCE_CANCELLED"] = 15
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
		_ = t.launchMetadataRecovery(delay, fetch)
	})
}

// RetryMetadataRecovery never re-adds the torrent or restarts the engine.
// It is explicit, rate-limited and subject to current source consent/policy.
func (t *Torrent) RetryMetadataRecovery() error {
	if t == nil || t.LTHandle() == nil || t.TorrentSpec == nil {
		return errors.New("torrent has no active metadata request")
	}
	return t.launchMetadataRecovery(0, fetchMetadataSource)
}

func (t *Torrent) launchMetadataRecovery(delay time.Duration, fetch func(context.Context, string) ([]byte, error)) error {
	t.mu.Lock()
	if t.metadataRecoveryRunning || time.Since(t.metadataRecoveryLast) < 30*time.Second {
		t.mu.Unlock()
		return errors.New("metadata recovery is running or was retried recently")
	}
	spec := *t.TorrentSpec
	spec.MetadataSources = append([]string(nil), spec.MetadataSources...)
	t.mu.Unlock()
	sources := metadataSources(&spec, settings.CurrentFlow())
	if len(sources) == 0 {
		return errors.New("no authorized metadata sources")
	}
	ctx, cancel := context.WithTimeout(context.Background(), delay+4*metadataSourceTimeout+time.Second)
	t.mu.Lock()
	if t.Stat == state.TorrentClosed {
		t.mu.Unlock()
		cancel()
		return errors.New("torrent is closed")
	}
	if t.metadataRecoveryRunning || time.Since(t.metadataRecoveryLast) < 30*time.Second {
		t.mu.Unlock()
		cancel()
		return errors.New("metadata recovery is running or was retried recently")
	}
	select {
	case <-t.gotInfoCh:
		t.mu.Unlock()
		cancel()
		return errors.New("metadata is already available")
	default:
	}
	t.metadataRecoveryRunning, t.metadataRecoveryLast = true, time.Now()
	t.metadataRecoveryCancel = cancel
	t.mu.Unlock()
	t.metadataRecoveryState(MetadataRecovery{State: "METADATA_WAIT"})
	go func() {
		defer cancel()
		defer func() { t.mu.Lock(); t.metadataRecoveryRunning = false; t.mu.Unlock() }()
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
	return nil
}

func (t *Torrent) recoverMetadata(ctx context.Context, sources []metadataSource, fetch func(context.Context, string) ([]byte, error)) {
	if t == nil || ctx.Err() != nil {
		return
	}
	handle := t.LTHandle()
	if handle == nil {
		return
	}
	if have, _ := handle.HaveMetadata(); have {
		t.signalGotInfo()
		return
	}
	if !settings.CurrentFlow().Enabled || !settings.CurrentFlow().MetadataFallback {
		t.metadataRecoveryState(MetadataRecovery{State: "METADATA_MISS", Code: "POLICY_CHANGED"})
		return
	}
	// Supplied sources retain preference: a public-cache tier starts only when
	// the authorized supplied tier is exhausted. Within each tier, hedge a slow
	// first source after one second, never more than two requests per torrent.
	for len(sources) > 0 && ctx.Err() == nil {
		end := 1
		for end < len(sources) && sources[end].kind == sources[0].kind {
			end++
		}
		if t.recoverMetadataTier(ctx, sources[:end], fetch) {
			return
		}
		sources = sources[end:]
	}
}

func metadataSourceAllowed(source metadataSource) bool {
	f := settings.CurrentFlow()
	return f.Enabled && f.MetadataFallback && (source.kind != "PUBLIC_CACHE" || f.PublicMetadataCache)
}

func (t *Torrent) recoverMetadataTier(parent context.Context, sources []metadataSource, fetch func(context.Context, string) ([]byte, error)) bool {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	type result struct {
		source metadataSource
		data   []byte
		err    error
	}
	results := make(chan result, 2)
	active, next := 0, 0
	policy := metadataFetchKey("")
	last := MetadataRecovery{State: "METADATA_MISS", Code: "NO_MATCHING_METADATA"}
	// Preserve the attempt count across supplied and public tiers.
	if s := t.FlowStartup().MetadataRecovery; s != nil {
		last.Attempts = s.Attempts
	}
	launch := func() {
		for next < len(sources) && active < 2 {
			source := sources[next]
			next++
			if !metadataSourceAllowed(source) {
				last.Source, last.State, last.Code = source.kind, "METADATA_MISS", "POLICY_CHANGED"
				t.metadataRecoveryState(last)
				continue
			}
			if !metadataHealth.Allowed(metadataServiceKey(source.url), time.Now()) {
				last.Source, last.Code = source.kind, "SOURCE_COOLDOWN"
				t.metadataRecoveryState(last)
				continue
			}
			last.Attempts++
			last.Source, last.State, last.Code = source.kind, "METADATA_HTTP", ""
			t.metadataRecoveryState(last)
			active++
			go func() {
				// A peer may have completed metadata between tier admission and
				// execution. Do not contact another source after that victory.
				handle := t.LTHandle()
				if handle != nil {
					if have, _ := handle.HaveMetadata(); have {
						t.signalGotInfo()
						results <- result{source: source, err: context.Canceled}
						return
					}
				}
				if ctx.Err() != nil || handle == nil || !metadataSourceAllowed(source) {
					results <- result{source: source, err: context.Canceled}
					return
				}
				data, err := fetch(ctx, source.url)
				select {
				case results <- result{source, data, err}:
				case <-ctx.Done():
				}
			}()
			break // the second request is staggered unless the first fails
		}
	}
	launch()
	hedge := time.NewTimer(time.Second)
	defer hedge.Stop()
	consent := time.NewTicker(100 * time.Millisecond)
	defer consent.Stop()
	for active > 0 {
		var response result
		select {
		case <-ctx.Done():
			last.State, last.Code = "METADATA_MISS", "SOURCE_CANCELLED"
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				last.Code = "SOURCE_TIMEOUT"
			}
			t.metadataRecoveryState(last)
			return true
		case <-t.gotInfoCh:
			return true
		case <-consent.C:
			if !metadataSourceAllowed(sources[0]) || metadataFetchKey("") != policy {
				last.State, last.Code = "METADATA_MISS", "POLICY_CHANGED"
				t.metadataRecoveryState(last)
				return true
			}
			continue
		case <-hedge.C:
			launch()
			continue
		case response = <-results:
			active--
		}
		if !metadataSourceAllowed(response.source) || metadataFetchKey("") != policy {
			last.State, last.Code = "METADATA_MISS", "POLICY_CHANGED"
			t.metadataRecoveryState(last)
			return true
		}
		handle := t.LTHandle()
		if handle == nil {
			return true
		}
		if have, _ := handle.HaveMetadata(); have {
			t.signalGotInfo()
			return true
		}
		data, err := response.data, response.err
		last.Source = response.source.kind
		last.Code = "SOURCE_NETWORK_ERROR"
		if errors.Is(err, flow.ErrFetchBusy) {
			last.Code = "FETCH_CAPACITY"
		}
		var sourceError metadataSourceError
		if errors.As(err, &sourceError) {
			last.Code = sourceError.code
		}
		if errors.Is(err, context.DeadlineExceeded) {
			last.Code = "SOURCE_TIMEOUT"
		}
		if ctx.Err() == nil && !errors.Is(err, context.Canceled) && !errors.Is(err, flow.ErrFetchBusy) {
			metadataHealth.Observe(metadataServiceKey(response.source.url), err != nil && (last.Code == "SOURCE_NETWORK_ERROR" || last.Code == "SOURCE_UNAVAILABLE" || last.Code == "SOURCE_TIMEOUT"), time.Now())
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
					return true
				}
				return true // a peer completed the same metadata concurrently
			}
		}
		last.State = "METADATA_MISS"
		t.metadataRecoveryState(last)
		launch()
	}
	return false
}
