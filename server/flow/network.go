package flow

import (
	"math/rand/v2"
	"net"
	"time"
)

// Jitter avoids synchronized retries while retaining configured hard bounds.
func RetryJitter(attempt, minimum, maximum int) time.Duration {
	if minimum < 1 || minimum > 60 {
		minimum = 2
	}
	if maximum < minimum || maximum > 600 {
		maximum = 60
	}
	d := RetryDelay(attempt, minimum, maximum)
	factor := 0.8 + rand.Float64()*0.4
	return min(time.Duration(maximum)*time.Second, max(time.Duration(minimum)*time.Second, time.Duration(float64(d)*factor)))
}

// RetryDelay bounds repeated network checks and external announce attempts.
func RetryDelay(attempt, minSeconds, maxSeconds int) time.Duration {
	if minSeconds < 1 || minSeconds > 60 {
		minSeconds = 2
	}
	if maxSeconds < minSeconds || maxSeconds > 600 {
		maxSeconds = 60
	}
	seconds := minSeconds
	for i := 0; i < attempt && seconds < maxSeconds; i++ {
		if seconds > maxSeconds/2 {
			seconds = maxSeconds
		} else {
			seconds *= 2
		}
	}
	return time.Duration(min(seconds, maxSeconds)) * time.Second
}

// UsableLocalIP filters out loopback, link-local and unspecified addresses.
// It describes local interface readiness, not confirmed Internet access.
func UsableLocalIP(ip net.IP) bool {
	return ip != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsUnspecified()
}
