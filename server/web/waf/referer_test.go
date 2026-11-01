package waf

import (
	"strings"
	"testing"
)

func TestHostMatchesBlocked(t *testing.T) {
	blocked := []string{"bylampa.online", "example.com"}

	tests := []struct {
		host string
		want bool
	}{
		{"bylampa.online", true},
		{"zerkalo.bylampa.online", true},
		{"www.bylampa.online", true},
		{"notbylampa.online", false},
		{"bylampa.online.evil.com", false},
		{"example.com", true},
		{"sub.example.com", true},
		{"localhost", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := hostMatchesBlocked(tt.host, blocked); got != tt.want {
			t.Errorf("hostMatchesBlocked(%q) = %v, want %v", tt.host, got, tt.want)
		}
	}
}

func TestIsBlockedReferer(t *testing.T) {
	blocked := []string{"bylampa.online"}

	if host, ok := isBlockedReferer("https://zerkalo.bylampa.online/player", "", blocked); !ok || host != "zerkalo.bylampa.online" {
		t.Fatalf("expected blocked referer, got host=%q ok=%v", host, ok)
	}
	if _, ok := isBlockedReferer("", "https://bylampa.online", blocked); !ok {
		t.Fatal("expected blocked origin")
	}
	if _, ok := isBlockedReferer("", "", blocked); ok {
		t.Fatal("expected empty headers to pass")
	}
	if _, ok := isBlockedReferer("https://myserver.local/stream", "", blocked); ok {
		t.Fatal("expected unrelated referer to pass")
	}
}

// MatriX.148 #946: rules and headers must identify the address, not its spelling.
func TestRefererRulesAndHeadersShareIPNormalization(t *testing.T) {
	for _, tc := range []struct {
		name, rules string
		headers     []string
		canonical   string
	}{
		{"IPv6", "2001:0db8::0001\n2001:db8:0:0:0:0:0:1\n", []string{
			"http://[2001:db8::1]/", "https://[2001:DB8:0:0:0:0:0:1]:8090/",
		}, "2001:db8::1"},
		{"mapped IPv4", "192.0.2.1\n::ffff:c000:201\n", []string{
			"http://192.0.2.1/", "http://[::ffff:192.0.2.1]:8090/",
		}, "192.0.2.1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blocked := scanRefererBuf([]byte(tc.rules))
			deduplicated := mergeReferers(nil, blocked)
			if len(deduplicated) != 1 || deduplicated[0] != tc.canonical {
				t.Fatalf("equivalent rules were not canonicalized/deduplicated: %q", deduplicated)
			}
			for _, header := range tc.headers {
				for _, origin := range []bool{false, true} {
					referer, originHeader := header, ""
					if origin {
						referer, originHeader = "", header
					}
					if host, ok := isBlockedReferer(referer, originHeader, deduplicated); !ok || host != tc.canonical {
						t.Errorf("referer=%q origin=%q: host=%q blocked=%v", referer, originHeader, host, ok)
					}
				}
			}
			if _, ok := isBlockedReferer("http://[2001:db8::2]/", "http://192.0.2.2/", deduplicated); ok {
				t.Fatal("another address was blocked")
			}
		})
	}
}

func TestBlockedReferersFromConfig(t *testing.T) {
	got := blockedReferersFromConfig([]byte("example.com\nbylampa.online\n"))
	want := append(append([]string(nil), defaultBlockedReferers...), "example.com")
	if len(got) != len(want) {
		t.Fatalf("blockedReferersFromConfig() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("blockedReferersFromConfig()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	empty := blockedReferersFromConfig(nil)
	if len(empty) != len(defaultBlockedReferers) {
		t.Fatalf("blockedReferersFromConfig(nil) = %v, want defaults %v", empty, defaultBlockedReferers)
	}
	for i, host := range defaultBlockedReferers {
		if empty[i] != host {
			t.Fatalf("blockedReferersFromConfig(nil)[%d] = %q, want %q", i, empty[i], host)
		}
	}
}

func TestScanRefererBuf(t *testing.T) {
	got := scanRefererBuf([]byte("# comment\nbylampa.online\n\n# another\nexample.com\n"))
	want := []string{"bylampa.online", "example.com"}
	if len(got) != len(want) {
		t.Fatalf("scanRefererBuf() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scanRefererBuf()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNormalizeRefererRule(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"Example.COM.", "example.com", true},
		{"example.com:443", "example.com", true},
		{"https://Example.COM/", "example.com", true},
		{"http://example.com:8080", "example.com", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"[2001:db8::1]:443", "2001:db8::1", true},
		{"https://example.com/path", "", false},
		{"ftp://example.com", "", false},
		{"*.example.com", "", false},
		{"example.com/path", "", false},
		{"-bad.example", "", false},
		{"bad_.example", "", false},
	}
	for _, tt := range tests {
		got, ok := normalizeRefererRule(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Errorf("normalizeRefererRule(%q) = %q, %v; want %q, %v", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}

func TestScanRefererBufWarnings(t *testing.T) {
	hosts, warnings := scanRefererBufValidated([]byte("example.com\n*.bad.example\nhttps://good.example/\n"))
	if len(hosts) != 2 || hosts[0] != "example.com" || hosts[1] != "good.example" {
		t.Fatalf("hosts=%v", hosts)
	}
	if len(warnings) != 1 || warnings[0].Line != 2 || warnings[0].Code != "invalid_referer" {
		t.Fatalf("warnings=%+v", warnings)
	}
}

func TestHostFromHeaderNormalizesPortAndTrailingDot(t *testing.T) {
	if got := hostFromHeader("https://Example.COM.:443/path"); got != "example.com" {
		t.Fatalf("hostFromHeader()=%q", got)
	}
}

func TestRefererRulesAndHeadersShareIDNNormalization(t *testing.T) {
	blocked := blockedReferersFromConfig([]byte("пример.рф\nxn--e1afmkfd.xn--p1ai\n"))
	count := 0
	for _, host := range blocked {
		if host == "xn--e1afmkfd.xn--p1ai" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("equivalent rules were not deduplicated: %q", blocked)
	}
	for _, tc := range []struct {
		url  string
		want bool
	}{
		{"https://пример.рф/", true},
		{"https://xn--e1afmkfd.xn--p1ai/", true},
		{"https://SUB.ПРИМЕР.РФ.:443/path", true},
		{"https://sub.xn--e1afmkfd.xn--p1ai/path", true},
		{"https://xn--e1afmkfd.xn--p1ai.evil.example/", false},
		{"https://notxn--e1afmkfd.xn--p1ai/", false},
		{"https://пример.рф@safe.example/", false},
	} {
		for _, origin := range []bool{false, true} {
			referer, originHeader := tc.url, ""
			if origin {
				referer, originHeader = "", tc.url
			}
			if _, got := isBlockedReferer(referer, originHeader, blocked); got != tc.want {
				t.Errorf("referer %q origin %q blocked=%v, want %v", referer, originHeader, got, tc.want)
			}
		}
	}
}

func TestIDNRulesRetainHostValidation(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		ok          bool
	}{
		{"https://BÜCHER.example/", "xn--bcher-kva.example", true},
		{"пример。рф。", "xn--e1afmkfd.xn--p1ai", true},
		{"bu\u0308cher.example", "xn--bcher-kva.example", true},
		{strings.Repeat("я", 60) + ".example", "", false},
		{"xn--.example", "", false},
		{"bad_.example", "", false},
		{"https://user@пример.рф/", "", false},
		{"https://пример.рф/path", "", false},
		{"*.пример.рф", "", false},
	} {
		if got, ok := normalizeRefererRule(tc.input); got != tc.want || ok != tc.ok {
			t.Errorf("normalizeRefererRule(%q) = %q, %v; want %q, %v", tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

func TestRefererScannerReportsOversizedLine(t *testing.T) {
	_, warnings := scanRefererBufValidated([]byte(strings.Repeat("a", maxScannerTokenSize+1)))
	if len(warnings) != 1 || warnings[0].Code != "line_too_long" {
		t.Fatalf("warnings=%+v", warnings)
	}
}
