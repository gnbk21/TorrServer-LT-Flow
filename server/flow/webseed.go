package flow

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func WebSeedID(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:16])
}

// DNS is resolved and filtered again by native code at every connection. This
// syntax check alone is not the SSRF boundary. LAN permission requires a literal
// address, forbids cloud link-local addresses, query arguments and credentials.
func ValidateWebSeed(value string, allowLocal bool) (*url.URL, error) {
	if len(value) > 8192 || strings.ContainsAny(value, "\r\n\x00") {
		return nil, errors.New("invalid mirror URL")
	}
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("HTTP(S) mirror URL required; credentials and fragments are not supported")
	}
	if port := u.Port(); port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return nil, errors.New("invalid mirror port")
		}
	}
	ip := net.ParseIP(u.Hostname())
	if allowLocal {
		if ip == nil || !(ip.IsPrivate() || ip.IsLoopback()) || ip.IsLinkLocalUnicast() || u.RawQuery != "" {
			return nil, errors.New("LAN mirror approval requires a private or loopback IP without query arguments")
		}
	} else if ip != nil && (!ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast()) {
		return nil, errors.New("local mirror requires explicit LAN approval")
	}
	return u, nil
}
