package torr

import (
	"errors"
	"net"
	"sort"
	"strings"
	"time"

	"server/flow"
	"server/lt"
	"server/netchange"
	"server/netpolicy"
	"server/settings"
	"server/torr/storage/torrstor"
)

// FlowNetworkStatus reports local address readiness. ADDRESS_READY does not
// assert that DNS, trackers or the Internet are reachable.
type FlowNetworkStatus struct {
	PeerTCPPort              int                `json:"peer_tcp_port"`
	PeerUDPPort              int                `json:"peer_udp_port"`
	MappedTCPPort            int                `json:"mapped_tcp_port"`
	MappedUDPPort            int                `json:"mapped_udp_port"`
	MappingSuccesses         uint64             `json:"mapping_successes"`
	MappingErrors            uint64             `json:"mapping_errors"`
	ListenerErrors           uint64             `json:"listener_errors"`
	IncomingTCP              uint64             `json:"incoming_tcp"`
	IncomingUTP              uint64             `json:"incoming_utp"`
	IncomingIPv6             uint64             `json:"incoming_ipv6"`
	MappingCheckedAt         time.Time          `json:"mapping_checked_at,omitempty"`
	DiagnosticHistory        flow.HistoryStatus `json:"diagnostic_history"`
	DHTStateRestored         bool               `json:"dht_state_restored"`
	StartedAt                time.Time          `json:"started_at"`
	AddressReadyMs           int64              `json:"address_ready_ms"`
	TransitionCount          uint64             `json:"transition_count"`
	RetryCount               uint64             `json:"retry_count"`
	LastCheckDurationMs      int64              `json:"last_check_duration_ms"`
	LastReannounceDurationMs int64              `json:"last_reannounce_duration_ms"`
	LastAddressRecoveryMs    int64              `json:"last_address_recovery_ms"`
	LastTrackerRecoveryMs    int64              `json:"last_tracker_recovery_ms"`
	State                    string             `json:"state"`
	Connectivity             string             `json:"connectivity"`
	Addresses                []string           `json:"addresses"`
	CheckedAt                time.Time          `json:"checked_at"`
	ChangedAt                time.Time          `json:"changed_at"`
	NextCheckSeconds         int                `json:"next_check_seconds"`
	ReannounceCount          uint64             `json:"reannounce_count"`
	LastTrackerReply         time.Time          `json:"last_tracker_reply,omitempty"`
	LastTrackerError         time.Time          `json:"last_tracker_error,omitempty"`
	ConnectivityLostAt       time.Time          `json:"connectivity_lost_at,omitempty"`
	LastError                string             `json:"last_error,omitempty"`
	NotificationCount        uint64             `json:"notification_count"`
	RecoveryReason           string             `json:"recovery_reason,omitempty"`
}

func (bt *BTServer) recordNetworkAlert(a *lt.Alert) {
	switch a.Type {
	case "listen_succeeded", "listen_succeeded_alert", "listen_failed", "listen_failed_alert", "portmap", "portmap_alert", "portmap_error", "portmap_error_alert", "incoming_connection", "incoming_connection_alert":
	default:
		return
	}
	bt.networkMu.Lock()
	defer bt.networkMu.Unlock()
	s := &bt.networkStatus
	switch a.Type {
	case "listen_succeeded", "listen_succeeded_alert":
		if a.Transport == "udp" {
			s.PeerUDPPort = a.Port
		} else {
			s.PeerTCPPort = a.Port
		}
	case "listen_failed", "listen_failed_alert":
		s.ListenerErrors++
	case "portmap", "portmap_alert":
		s.MappingSuccesses++
		s.MappingCheckedAt = time.Now()
		if a.Transport == "tcp" {
			s.MappedTCPPort = a.Port
		} else {
			s.MappedUDPPort = a.Port
		}
	case "portmap_error", "portmap_error_alert":
		s.MappingErrors++
		s.MappingCheckedAt = time.Now()
	case "incoming_connection", "incoming_connection_alert":
		if a.Transport == "utp" {
			s.IncomingUTP++
		} else {
			s.IncomingTCP++
		}
		if a.IPv6 {
			s.IncomingIPv6++
		}
	}
}

type networkTracker struct {
	fingerprint  string
	needAnnounce bool
	ready        bool
}

func (t *networkTracker) observe(addresses []string, err error) bool {
	ready := len(addresses) > 0 && err == nil
	fingerprint := strings.Join(addresses, ",")
	if fingerprint != t.fingerprint || ready != t.ready {
		t.fingerprint = fingerprint
		t.needAnnounce = ready
	}
	t.ready = ready
	return ready
}

func (bt *BTServer) FlowNetworkStatus() FlowNetworkStatus {
	if bt == nil {
		return FlowNetworkStatus{State: "NOT_STARTED"}
	}
	bt.networkMu.Lock()
	defer bt.networkMu.Unlock()
	s := bt.networkStatus
	s.DiagnosticHistory = bt.history.Load().Status()
	s.DHTStateRestored = bt.dhtRestored.Load()
	s.Addresses = append([]string(nil), s.Addresses...)
	if s.State == "" {
		s.State = "UNKNOWN"
	}
	if s.Connectivity == "" {
		s.Connectivity = "INTERNET_WAIT"
	}
	return s
}

// A tracker reply proves that at least one external tracker transport worked.
// An error does not imply every tracker is down, so it marks a previously
// working network as degraded rather than claiming complete Internet loss.
func (bt *BTServer) recordTrackerConnectivity(alertType string) {
	if bt == nil {
		return
	}
	bt.networkMu.Lock()
	defer bt.networkMu.Unlock()
	if bt.networkStatus.State != "ADDRESS_READY" {
		return
	}
	now := time.Now()
	switch alertType {
	case "tracker_reply", "tracker_reply_alert":
		if bt.networkStatus.Connectivity != "ONLINE" && !bt.networkStatus.ConnectivityLostAt.IsZero() {
			bt.networkStatus.LastTrackerRecoveryMs = now.Sub(bt.networkStatus.ConnectivityLostAt).Milliseconds()
		}
		bt.networkStatus.Connectivity = "ONLINE"
		bt.networkStatus.ConnectivityLostAt = time.Time{}
		bt.networkStatus.LastTrackerReply = now
	case "tracker_error", "tracker_error_alert":
		if bt.networkStatus.Connectivity == "ONLINE" {
			bt.networkStatus.Connectivity = "DEGRADED"
			bt.networkStatus.ConnectivityLostAt = now
		}
		bt.networkStatus.LastTrackerError = now
	}
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

func (bt *BTServer) reannounceOnNetworkChange(stop <-chan struct{}) (int, error) {
	count := 0
	for _, tor := range bt.ListTorrents() {
		select {
		case <-stop:
			return count, nil
		default:
		}
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
		state, stateErr := handle.Status()
		if s := settings.BTsets(); stateErr == nil && state.HasMetadata && !state.Private && (s == nil || !s.DisableDHT) {
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
	events, closeWatch := netchange.Watch()
	defer closeWatch()
	var lastCheck, lastAnnounce time.Time
	notified := false
	defer func() { bt.networkMu.Lock(); bt.networkStatus.State = "STOPPED"; bt.networkMu.Unlock() }()
	var tracker networkTracker
	var interfaceFingerprint string
	attempt := 0
	started := time.Now()
	for {
		select {
		case <-stop:
			return
		default:
		}
		checkStarted := time.Now()
		addresses, err := localNetworkAddresses()
		if policyErr := bt.reconcileInterfacePolicy(&interfaceFingerprint); policyErr != nil {
			err = policyErr
		}
		if policy := netpolicy.Snapshot(); policy.Name != "" && policy.State != "BOUND" {
			err = errors.New("selected torrent interface is unavailable")
		}
		now := time.Now()
		resumed := !lastCheck.IsZero() && now.Sub(lastCheck) > 90*time.Second
		lastCheck = now
		ready := tracker.observe(addresses, err)
		torrstor.Global().SetNetworkRecovering(!ready)
		if (notified || resumed) && now.Sub(lastAnnounce) >= 30*time.Second {
			tracker.needAnnounce = true
		}
		announced := 0
		var announceDuration time.Duration
		if ready && tracker.needAnnounce && (lastAnnounce.IsZero() || now.Sub(lastAnnounce) >= 30*time.Second) {
			announceStarted := time.Now()
			announced, err = bt.reannounceOnNetworkChange(stop)
			announceDuration = time.Since(announceStarted)
			if err == nil {
				tracker.needAnnounce = false
				lastAnnounce = now
			}
		}
		f := settings.CurrentFlow()
		wait := 15 * time.Second
		if !ready || err != nil {
			wait = flow.RetryJitter(attempt, f.NetworkRetryMinSec, f.NetworkRetryMaxSec)
			attempt++
		} else {
			attempt = 0
		}
		bt.networkMu.Lock()
		status := &bt.networkStatus
		if status.StartedAt.IsZero() {
			status.StartedAt = started
		}
		if ready && status.AddressReadyMs == 0 {
			status.AddressReadyMs = max(int64(1), now.Sub(started).Milliseconds())
		}
		if attempt > 0 {
			status.RetryCount++
		}
		status.LastCheckDurationMs = time.Since(checkStarted).Milliseconds()
		if notified {
			status.NotificationCount++
			status.RecoveryReason = "INTERFACE_OR_ROUTE_CHANGE"
		}
		if resumed {
			status.RecoveryReason = "RESUME_OR_DELAYED_CHECK"
		}
		notified = false
		if announceDuration > 0 {
			status.LastReannounceDurationMs = announceDuration.Milliseconds()
		}
		state := "NO_ADDRESS"
		if ready {
			state = "ADDRESS_READY"
		}
		addressChanged := !sameAddresses(status.Addresses, addresses)
		if status.State != state || addressChanged {
			if ready && status.State == "NO_ADDRESS" && !status.ChangedAt.IsZero() {
				status.LastAddressRecoveryMs = now.Sub(status.ChangedAt).Milliseconds()
			}
			status.TransitionCount++
			status.ChangedAt = now
		}
		if !ready || addressChanged {
			status.Connectivity = "INTERNET_WAIT"
			if status.ConnectivityLostAt.IsZero() {
				status.ConnectivityLostAt = now
			}
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
		case <-events:
			timer.Stop()
			// Debounce notification bursts and never block callback cancellation.
			debounce := time.NewTimer(750 * time.Millisecond)
			select {
			case <-stop:
				debounce.Stop()
				return
			case <-debounce.C:
			}
			notified = true
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
