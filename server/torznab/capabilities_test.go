package torznab

import (
	"fmt"
	"server/searchstatus"
	"testing"
	"time"
)

func TestStructuredQueryUsesOnlyAdvertisedParameters(t *testing.T) {
	caps, err := ParseCapsBytes([]byte(`<caps><searching><movie-search available="yes" supportedParams="q,imdbid,year"/><tv-search available="yes" supportedParams="q,tvdbid,season,ep"/></searching></caps>`))
	if err != nil {
		t.Fatal(err)
	}
	query := Query{Text: "Generated", Type: "movie", IMDb: "tt1234567", TMDB: "9", Year: "2026"}
	params, mode := planQuery(query, caps)
	if mode != "movie" || params.Get("t") != "movie" || params.Get("imdbid") != "1234567" || params.Has("tmdbid") {
		t.Fatal(params, mode)
	}
	params, mode = planQuery(Query{Text: "Generated", Type: "tv", TVDB: "42", Season: "0", Episode: "2"}, caps)
	if params.Get("t") != "tvsearch" || params.Get("season") != "0" || mode != "tv" {
		t.Fatal(params, mode)
	}
	if _, mode = planQuery(query, nil); mode != "generic" {
		t.Fatal("cold request waited for caps")
	}
}
func TestStructuredValidationAndCredentialInvalidation(t *testing.T) {
	for _, q := range []Query{{Text: "ok", Type: "bad"}, {IMDb: "tt"}, {Text: "ok", Season: "-1"}, {Text: "ok", IMDb: "123&apikey=oops"}} {
		if q.Validate() == nil {
			t.Fatal("accepted invalid query", q)
		}
	}
	host := "https://example.invalid/" + fmt.Sprint(time.Now().UnixNano())
	rememberCaps(host, "old-secret", &Caps{Searching: Searching{Movie: SearchCapability{Available: "yes", SupportedParams: "imdbid"}}})
	capabilityCache.Lock()
	_, same := capabilityCache.entries[searchstatus.EndpointKey("torznab", host, "new-secret")]
	capabilityCache.Unlock()
	if same {
		t.Fatal("credentials reused capability identity")
	}
}
