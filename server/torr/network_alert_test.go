package torr

import (
	"server/lt"
	"testing"
)

func TestNativeConnectivityEvidenceIsDistinct(t *testing.T) {
	bt := NewBTS()
	for _, a := range []lt.Alert{{Type: "listen_succeeded", Port: 51413, Transport: "tcp"}, {Type: "listen_succeeded", Port: 51413, Transport: "udp"}, {Type: "portmap", Port: 62000, Transport: "tcp"}, {Type: "portmap_error", ErrorCode: 1}, {Type: "incoming_connection", Transport: "utp", IPv6: true}} {
		bt.recordNetworkAlert(&a)
	}
	s := bt.FlowNetworkStatus()
	if s.PeerTCPPort != 51413 || s.PeerUDPPort != 51413 || s.MappedTCPPort != 62000 || s.MappedUDPPort != 0 || s.MappingSuccesses != 1 || s.MappingErrors != 1 || s.IncomingUTP != 1 || s.IncomingIPv6 != 1 {
		t.Fatalf("%+v", s)
	}
	if s.Connectivity == "ONLINE" {
		t.Fatal("local listener or mapping claimed Internet connectivity")
	}
}
