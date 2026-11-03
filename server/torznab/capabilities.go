package torznab

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"server/rutor/models"
	"server/searchstatus"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SearchCapability struct {
	Available       string `xml:"available,attr" json:"available"`
	SupportedParams string `xml:"supportedParams,attr" json:"supported_params"`
}
type Searching struct {
	Search SearchCapability `xml:"search" json:"search"`
	TV     SearchCapability `xml:"tv-search" json:"tv"`
	Movie  SearchCapability `xml:"movie-search" json:"movie"`
}
type Query struct{ Text, Type, IMDb, TMDB, TVDB, Year, Season, Episode string }

func (q Query) Validate() error {
	if len(q.Text) > 1024 || (q.Type != "" && q.Type != "search" && q.Type != "movie" && q.Type != "tv") {
		return fmt.Errorf("invalid search query")
	}
	if q.IMDb != "" && strings.TrimPrefix(q.IMDb, "tt") == "" {
		return fmt.Errorf("invalid identifier")
	}
	for _, id := range []string{strings.TrimPrefix(q.IMDb, "tt"), q.TMDB, q.TVDB} {
		if len(id) > 12 {
			return fmt.Errorf("invalid identifier")
		}
		for _, c := range id {
			if c < '0' || c > '9' {
				return fmt.Errorf("invalid identifier")
			}
		}
	}
	for _, value := range []struct {
		s      string
		lo, hi int
	}{{q.Year, 1870, 2100}, {q.Season, 0, 9999}, {q.Episode, 0, 99999}} {
		if value.s != "" {
			n, err := strconv.Atoi(value.s)
			if err != nil || n < value.lo || n > value.hi {
				return fmt.Errorf("invalid year or episode")
			}
		}
	}
	if strings.TrimSpace(q.Text) == "" && q.IMDb == "" && q.TMDB == "" && q.TVDB == "" {
		return fmt.Errorf("a title or identifier is required")
	}
	return nil
}

type capabilityEntry struct {
	caps    *Caps
	expires time.Time
	pending bool
}

var capabilityCache = struct {
	sync.Mutex
	entries map[string]capabilityEntry
}{entries: map[string]capabilityEntry{}}

// Read-only presentation must never trigger capability discovery.
func PeekCapabilities(host, key string) *Searching {
	id := searchstatus.EndpointKey("torznab", host, key)
	capabilityCache.Lock()
	defer capabilityCache.Unlock()
	entry := capabilityCache.entries[id]
	if entry.caps == nil || !time.Now().Before(entry.expires) {
		return nil
	}
	copy := entry.caps.Searching
	return &copy
}

func rememberCaps(host, key string, caps *Caps) {
	id := searchstatus.EndpointKey("torznab", host, key)
	capabilityCache.Lock()
	defer capabilityCache.Unlock()
	if len(capabilityCache.entries) >= 128 {
		for k := range capabilityCache.entries {
			if k != id {
				delete(capabilityCache.entries, k)
				break
			}
		}
	}
	copy := *caps
	copy.Categories = nil
	for _, value := range []*SearchCapability{&copy.Searching.Search, &copy.Searching.TV, &copy.Searching.Movie} {
		value.Available = strings.ToLower(strings.TrimSpace(value.Available))
		value.SupportedParams = value.SupportedParams[:min(512, len(value.SupportedParams))]
	}
	capabilityCache.entries[id] = capabilityEntry{caps: &copy, expires: time.Now().Add(15 * time.Minute)}
}
func FetchCaps(ctx context.Context, host, key string) (*Caps, error) {
	caps, err := fetchCaps(ctx, host, key)
	if err == nil {
		rememberCaps(host, key, caps)
	}
	return caps, err
}

// A cold generic search never waits for capability discovery. One bounded
// background fetch shares the same global network admission as normal search.
func cachedCapabilities(host, key string) *Caps {
	id := searchstatus.EndpointKey("torznab", host, key)
	capabilityCache.Lock()
	entry, ok := capabilityCache.entries[id]
	if ok && time.Now().Before(entry.expires) {
		capabilityCache.Unlock()
		return entry.caps
	}
	if len(capabilityCache.entries) >= 128 {
		for k := range capabilityCache.entries {
			delete(capabilityCache.entries, k)
			break
		}
	}
	capabilityCache.entries[id] = capabilityEntry{pending: true, expires: time.Now().Add(10 * time.Second)}
	capabilityCache.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		caps, err := fetchCaps(ctx, host, key)
		if err == nil {
			rememberCaps(host, key, caps)
		} else {
			capabilityCache.Lock()
			if e := capabilityCache.entries[id]; e.pending {
				capabilityCache.entries[id] = capabilityEntry{expires: time.Now().Add(time.Minute)}
			}
			capabilityCache.Unlock()
		}
	}()
	return nil
}

var errUnsupported = errors.New("unsupported structured query")

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
func planQuery(q Query, caps *Caps) (url.Values, string) {
	generic := url.Values{"t": {"search"}, "q": {q.Text}}
	if caps == nil {
		return generic, "generic"
	}
	kind := q.Type
	if kind == "" {
		if q.TVDB != "" || q.Season != "" || q.Episode != "" {
			kind = "tv"
		} else if q.IMDb != "" || q.TMDB != "" {
			kind = "movie"
		}
	}
	var cap SearchCapability
	switch kind {
	case "movie":
		cap = caps.Searching.Movie
	case "tv":
		cap = caps.Searching.TV
	default:
		kind = "search"
		cap = caps.Searching.Search
	}
	if cap.Available != "yes" && cap.Available != "true" && cap.Available != "1" {
		kind = "search"
		cap = caps.Searching.Search
		if cap.Available != "yes" && cap.Available != "true" && cap.Available != "1" {
			return generic, "generic"
		}
	}
	supported := map[string]bool{}
	for _, p := range strings.Split(strings.ToLower(cap.SupportedParams), ",") {
		supported[strings.TrimSpace(p)] = true
	}
	values := url.Values{"t": {kind}}
	if kind == "tv" {
		values.Set("t", "tvsearch")
	}
	for key, value := range map[string]string{"q": q.Text, "imdbid": strings.TrimPrefix(q.IMDb, "tt"), "tmdbid": q.TMDB, "tvdbid": q.TVDB, "year": q.Year, "season": q.Season, "ep": q.Episode} {
		if value != "" && supported[key] {
			values.Set(key, value)
		}
	}
	if len(values) == 1 {
		return generic, "generic"
	}
	if kind == "search" {
		return values, "generic"
	}
	return values, kind
}
func searchPlanned(ctx context.Context, host, key string, q Query, label, cat string, offset, limit int) ([]*models.TorrentDetails, string, error) {
	if until := searchstatus.History(searchstatus.EndpointKey("torznab", host, key)).CooldownUntil; until != nil && time.Now().Before(*until) {
		return nil, "cooldown", searchstatus.HTTP(429)
	}
	var caps *Caps
	if cached := PeekCapabilities(host, key); cached != nil {
		caps = &Caps{Searching: *cached}
	}
	params, mode := planQuery(q, caps)
	if strings.TrimSpace(params.Get("q")) == "" && params.Get("imdbid") == "" && params.Get("tmdbid") == "" && params.Get("tvdbid") == "" {
		return nil, mode, errUnsupported
	}
	rows, err := searchOneRequest(ctx, host, key, q.Text, label, cat, offset, limit, params)
	structured := mode != "generic"
	for name := range params {
		if name != "t" && name != "q" {
			structured = true
		}
	}
	if structured && strings.TrimSpace(q.Text) != "" && (errors.Is(err, errUnsupported) || (err == nil && len(rows) == 0)) {
		rows, err = searchOneRequest(ctx, host, key, q.Text, label, cat, offset, limit, nil)
		mode = "generic_fallback"
	}
	return rows, mode, err
}
