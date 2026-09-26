package torr

import (
	"net"
	"sort"
	"strings"
	"time"

	"server/flow"
	"server/settings"
)

// FlowNetworkStatus reports local address readiness. ADDRESS_READY does not
// assert that DNS, trackers or the Internet are reachable.
type FlowNetworkStatus struct {
	State            string    `json:"state"`
	Addresses        []string  `json:"addresses"`
	CheckedAt        time.Time `json:"checked_at"`
	ChangedAt        time.Time `json:"changed_at"`
	NextCheckSeconds int       `json:"next_check_seconds"`
	ReannounceCount  uint64    `json:"reannounce_count"`
	LastError        string    `json:"last_error,omitempty"`
}

type networkTracker struct {
	fingerprint  string
	needAnnounce bool
}

func (t *networkTracker) observe(addresses []string, err error) bool {
	ready := len(addresses) > 0 && err == nil
	fingerprint := strings.Join(addresses, ",")
	if fingerprint != t.fingerprint {
		t.fingerprint = fingerprint
		t.needAnnounce = ready
	}
	return ready
}

func (bt *BTServer) FlowNetworkStatus() FlowNetworkStatus {
	if bt == nil {
		return FlowNetworkStatus{State: "NOT_STARTED"}
	}
	bt.networkMu.Lock()
	defer bt.networkMu.Unlock()
	s := bt.networkStatus
	s.Addresses = append([]string(nil), s.Addresses...)
	if s.State == "" {
		s.State = "UNKNOWN"
	}
	return s
}

func NetworkStatusSnapshot() FlowNetworkStatus {
	if bts == nil {
		return FlowNetworkStatus{State: "NOT_STARTED"}
	}
	return bts.FlowNetworkStatus()
}

func localNetworkAddresses() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			var ip net.IP
			switch a := address.(type) {
			case *net.IPNet:
				ip = a.IP
			case *net.IPAddr:
				ip = a.IP
			}
			if flow.UsableLocalIP(ip) {
				seen[ip.String()] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(seen))
	for address := range seen {
		out = append(out, address)
	}
	sort.Strings(out)
	return out, nil
}

func (bt *BTServer) reannounceOnNetworkChange() (int, error) {
	count := 0
	for _, tor := range bt.ListTorrents() {
		if tor == nil {
			continue
		}
		handle := tor.LTHandle()
		if handle == nil {
			continue
		}
		if err := handle.ForceReannounce(); err != nil {
			return count, err
		}
		if s := settings.BTsets(); s == nil || !s.DisableDHT {
			if err := handle.ForceDhtAnnounce(); err != nil {
				return count, err
			}
		}
		count++
	}
	return count, nil
}

// networkLifecycle polls local addresses as a fallback to unreliable or missed
// Windows notifications. It never blocks startup on an external host.
func (bt *BTServer) networkLifecycle(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	var tracker networkTracker
	attempt := 0
	for {
		addresses, err := localNetworkAddresses()
		now := time.Now()
		ready := tracker.observe(addresses, err)
		announced := 0
		if ready && tracker.needAnnounce {
			announced, err = bt.reannounceOnNetworkChange()
			if err == nil {
				tracker.needAnnounce = false
			}
		}
		f := settings.CurrentFlow()
		wait := 15 * time.Second
		if !ready || err != nil {
			wait = flow.RetryDelay(attempt, f.NetworkRetryMinSec, f.NetworkRetryMaxSec)
			attempt++
		} else {
			attempt = 0
		}
		bt.networkMu.Lock()
		status := &bt.networkStatus
		state := "NO_ADDRESS"
		if ready {
			state = "ADDRESS_READY"
		}
		if status.State != state || !sameAddresses(status.Addresses, addresses) {
			status.ChangedAt = now
		}
		status.State, status.Addresses, status.CheckedAt = state, addresses, now
		status.NextCheckSeconds = int(wait / time.Second)
		status.ReannounceCount += uint64(announced)
		status.LastError = ""
		if err != nil {
			status.LastError = err.Error()
		}
		bt.networkMu.Unlock()
		timer := time.NewTimer(wait)
		select {
		case <-stop:
			timer.Stop()
			bt.networkMu.Lock()
			bt.networkStatus.State = "STOPPED"
			bt.networkMu.Unlock()
			return
		case <-timer.C:
		}
	}
}

func sameAddresses(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
