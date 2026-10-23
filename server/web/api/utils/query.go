package utils

import (
	"net/url"
	"strings"
)

// DecodeQueryValue accepts the extra percent-encoding used by older clients
// after net/http has decoded the outer query. A literal plus is already a plus
// at this boundary, not another form-space. Malformed escapes remain intact.
func DecodeQueryValue(value string) string {
	decoded, err := url.PathUnescape(value)
	if err != nil {
		return value
	}
	return decoded
}

// DecodeQueryLink unwraps a legacy encoded link once, without decoding the
// components of an already recognizable URL. Otherwise a signed URL or magnet
// tracker can change identity before parsing. Callers validate this exact result.
func DecodeQueryLink(value string) string {
	value = strings.TrimSpace(value)
	if parsed, err := url.Parse(value); err == nil && parsed.Scheme != "" && parsed.Opaque == "" {
		return value
	}
	return strings.TrimSpace(DecodeQueryValue(value))
}
