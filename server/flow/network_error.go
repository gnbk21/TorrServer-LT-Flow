package flow

import "runtime"

func SafeTransport(value string) string {
	if value == "tcp" || value == "utp" {
		return value
	}
	return "unknown"
}

func SafeAddressFamily(value string) string {
	if value == "IPV4" || value == "IPV6" {
		return value
	}
	return "UNKNOWN"
}

// Error categories are deliberately closed: native names and exception text
// must not become unbounded labels or disclose remote destinations.
func SafeErrorCategory(category string) string {
	switch category {
	case "SYSTEM", "GENERIC", "DNS", "ASIO", "TLS", "HTTP", "LIBTORRENT":
		return category
	default:
		return "UNKNOWN"
	}
}

func NetworkErrorClass(category string, code int) string {
	switch SafeErrorCategory(category) {
	case "DNS":
		return "DNS"
	case "TLS":
		return "TLS"
	case "HTTP":
		if code >= 400 && code < 500 {
			return "HTTP_4XX"
		}
		if code >= 500 && code < 600 {
			return "HTTP_5XX"
		}
	case "SYSTEM", "GENERIC":
		// This classifier is for live alerts produced on this process's OS.
		// Retained cross-platform reports preserve numbers without guessing.
		switch code {
		case 10060:
			return "TIMEOUT"
		case 10061:
			return "REFUSED"
		case 10051, 10065:
			return "UNREACHABLE"
		}
		if runtime.GOOS == "linux" || runtime.GOOS == "android" {
			switch code {
			case 110:
				return "TIMEOUT"
			case 111:
				return "REFUSED"
			case 101, 113:
				return "UNREACHABLE"
			}
		}
		if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
			switch code {
			case 60:
				return "TIMEOUT"
			case 61:
				return "REFUSED"
			case 51, 65:
				return "UNREACHABLE"
			}
		}
	}
	return "UNKNOWN"
}
