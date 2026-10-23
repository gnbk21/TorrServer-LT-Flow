package utils

import (
	"net/url"
	"strings"
	"testing"
)

func TestDecodeQueryValuesAfterOuterQuery(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"Plain Title", "Plain Title"}, {"100% Pure", "100% Pure"},
		{"50%", "50%"}, {"%zz", "%zz"}, {"C++", "C++"},
		{"Caf%C3%A9", "Café"}, {"Сериал + кино", "Сериал + кино"},
		{"A%20B%2BC", "A B+C"}, {"%252F", "%2F"}, {"", ""},
	} {
		t.Run(tc.input, func(t *testing.T) {
			outer := url.Values{"query": {tc.input}}.Encode()
			values, err := url.ParseQuery(outer)
			if err != nil {
				t.Fatal(err)
			}
			if got := DecodeQueryValue(values.Get("query")); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDecodeQueryLinkPreservesComponentsAndLegacyWrapper(t *testing.T) {
	hash := strings.Repeat("a", 40)
	for _, link := range []string{
		"https://source.example/a%2Fb.torrent?token=A%2BB+Z&sig=%252F",
		"magnet:?xt=urn:btih:" + hash + "&dn=100%25+Pure&tr=https%3A%2F%2Ftracker.example%2Fannounce%3Fkey%3DA%252BB",
		"file:///C:/Media/a%20b.torrent", "torrs://abc123", hash,
	} {
		for _, input := range []string{link, url.PathEscape(link), "  " + link + "  "} {
			values, err := url.ParseQuery(url.Values{"link": {input}}.Encode())
			if err != nil {
				t.Fatal(err)
			}
			if got := DecodeQueryLink(values.Get("link")); got != link {
				t.Fatalf("changed link components: got %q, want %q", got, link)
			}
		}
	}
}

func TestCanonicalLinkAuthorization(t *testing.T) {
	for _, value := range []string{
		"https://source.example/a%2Fb.torrent?token=A%2BB", "file:///C:/state/test.torrent",
	} {
		for _, input := range []string{value, url.PathEscape(value), "  " + value + "  "} {
			link := DecodeQueryLink(input)
			if TestLink(link, false) == nil {
				t.Fatalf("unauthenticated source accepted: %q", input)
			}
			if err := TestLink(link, true); err != nil {
				t.Fatal(err)
			}
		}
	}
	magnet := "magnet:?xt=urn:btih:" + strings.Repeat("a", 40) + "&dn=100%25+Pure"
	if err := TestLink(DecodeQueryLink(url.PathEscape(magnet)), false); err != nil {
		t.Fatal(err)
	}
	if DecodeQueryLink("https%253A%252F%252Fsource.example") != "https%3A%2F%2Fsource.example" {
		t.Fatal("recursively decoded a source")
	}
}
