package torr

import (
	"errors"
	"testing"
)

func TestNetworkTrackerRecoversAfterAddressChanges(t *testing.T) {
	var tracker networkTracker
	if tracker.observe(nil, nil) || tracker.needAnnounce {
		t.Fatal("offline startup should not announce")
	}
	if !tracker.observe([]string{"192.168.1.10"}, nil) || !tracker.needAnnounce {
		t.Fatal("new address should request an announce")
	}
	tracker.needAnnounce = false // successful announce
	tracker.observe([]string{"192.168.1.10"}, nil)
	if tracker.needAnnounce {
		t.Fatal("unchanged address should not repeat an announce")
	}
	tracker.observe([]string{"192.168.1.20"}, nil)
	if !tracker.needAnnounce {
		t.Fatal("address change should request an announce")
	}
	tracker.observe(nil, errors.New("interfaces unavailable"))
	if tracker.needAnnounce {
		t.Fatal("offline state should not retain an announce request")
	}
	tracker.observe([]string{"192.168.1.20"}, nil)
	if !tracker.needAnnounce {
		t.Fatal("recovery should request an announce")
	}
}

func TestTrackerConnectivityRequiresTransportReply(t *testing.T) {
	bt := &BTServer{networkStatus: FlowNetworkStatus{State: "ADDRESS_READY", Connectivity: "INTERNET_WAIT"}}
	bt.recordTrackerConnectivity("tracker_error")
	if got := bt.FlowNetworkStatus().Connectivity; got != "INTERNET_WAIT" {
		t.Fatalf("tracker error before success = %q", got)
	}
	bt.recordTrackerConnectivity("tracker_reply")
	if got := bt.FlowNetworkStatus().Connectivity; got != "ONLINE" {
		t.Fatalf("tracker reply = %q", got)
	}
	bt.recordTrackerConnectivity("tracker_error")
	if got := bt.FlowNetworkStatus().Connectivity; got != "DEGRADED" {
		t.Fatalf("tracker error after success = %q", got)
	}
	bt.recordTrackerConnectivity("tracker_reply")
	if got := bt.FlowNetworkStatus().Connectivity; got != "ONLINE" {
		t.Fatalf("recovery reply = %q", got)
	}
	bt.networkStatus.State = "NO_ADDRESS"
	bt.networkStatus.Connectivity = "INTERNET_WAIT"
	bt.recordTrackerConnectivity("tracker_reply")
	if got := bt.FlowNetworkStatus().Connectivity; got != "INTERNET_WAIT" {
		t.Fatalf("reply without address changed connectivity to %q", got)
	}
}
