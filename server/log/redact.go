package log

import (
	"net/url"
	"regexp"
	"strings"
)

var remoteURL = regexp.MustCompile(`(?i)(?:https?|wss?|udp|socks(?:4a?|5h?)|magnet|torrs)://[^\s<>"']+|magnet:\?[^\s<>"']+`)
var sensitiveValue = regexp.MustCompile(`(?i)\b((?:probe_key|passkey|password|passwd|api[_-]?key|(?:access|session|client|bot|tg)[_-]?token|token|ss|authorization|secret|capability)["']?\s*[=:]\s*)(?:"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|[^\s&;,"']+)`)
var authorizationValue = regexp.MustCompile(`(?i)(authorization["']?\s*[=:]\s*)(?:bearer|basic)\s+[^\s&,;"']+`)
var capabilityPath = regexp.MustCompile(`/flow/play/[^\s/?"']+`)

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
	s = authorizationValue.ReplaceAllString(s, "${1}[redacted]")
	return sensitiveValue.ReplaceAllString(s, "${1}[redacted]")
}
