package log

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var remoteURL = regexp.MustCompile(`(?i)(?:https?|wss?|udp|socks(?:4a?|5h?)|magnet|torrs)://[^\s<>"']+|magnet:\?[^\s<>"']+`)
var sensitiveValue = regexp.MustCompile(`(?i)\b((?:probe_key|passkey|password|passwd|api[_-]?key|(?:access|session|client|bot|tg)[_-]?token|token|ss|authorization|secret|capability)["']?\s*[=:]\s*)(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s&;,"']+)`)
var authorizationValue = regexp.MustCompile(`(?i)(authorization["']?\s*[=:]\s*)(?:bearer|basic)\s+[^\s&,;"']+`)
var capabilityPath = regexp.MustCompile(`/flow/play/[^\s/?"']+`)
var pairingPath = regexp.MustCompile(`/flow/pairing/(?:proof|status)/[^\s/?"']+`)

// RedactSecrets strips all URL paths/queries: tracker secrets may be embedded in
// arbitrary path components rather than recognizable query parameter names.
func RedactSecrets(s string) string {
	s = remoteURL.ReplaceAllStringFunc(s, func(raw string) string {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return "[redacted URL]"
		}
		return strings.ToLower(u.Scheme) + "://" + u.Host + "/[redacted]"
	})
	s = capabilityPath.ReplaceAllString(s, "/flow/play/[redacted]")
	s = pairingPath.ReplaceAllString(s, "/flow/pairing/[redacted]")
	s = authorizationValue.ReplaceAllString(s, "${1}[redacted]")
	return sensitiveValue.ReplaceAllString(s, "${1}[redacted]")
}

// Each caller creates one timestamped record. Escape embedded line separators
// and terminal controls so remote titles/errors cannot forge another record.
func safeLogEntry(s string) string {
	s = strings.TrimSuffix(s, "\n")
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	var output strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' ||
			(r >= '\u202a' && r <= '\u202e') || (r >= '\u2066' && r <= '\u2069') {
			escaped := strconv.QuoteRune(r)
			output.WriteString(escaped[1 : len(escaped)-1])
		} else {
			output.WriteRune(r)
		}
	}
	return output.String()
}
