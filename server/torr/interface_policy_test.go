package torr

import (
	"server/lt"
	"server/netpolicy"
	"server/settings"
	"testing"
)

func TestInterfacePolicyUnavailableAndFamilyRecovery(t *testing.T) {
	s := settings.NewDefaultConfig()
	s.PeersListenPort = 51234
	cfg := lt.SessionConfig{}
	applyInterfacePolicy(cfg, netpolicy.InterfaceStatus{Name: "fixture", State: "WAIT_INTERFACE"}, s)
	if cfg["tsl_network_paused"] != true || cfg["outgoing_interfaces"] != "127.0.0.1" || cfg["enable_dht"] != false || cfg["enable_outgoing_tcp"] != false || cfg["enable_outgoing_utp"] != false {
		t.Fatal("missing interface did not fail closed", cfg)
	}
	cfg = lt.SessionConfig{}
	applyInterfacePolicy(cfg, netpolicy.InterfaceStatus{Name: "fixture", State: "BOUND", Addresses: []string{"192.0.2.12"}}, s)
	if cfg["tsl_network_paused"] != false || cfg["listen_interfaces"] != "192.0.2.12:51234" || cfg["outgoing_interfaces"] != "192.0.2.12" || cfg["enable_upnp"] != false || cfg["enable_natpmp"] != false || cfg["enable_lsd"] != false {
		t.Fatal(cfg)
	}
	s.EnableIPv6 = false
	applyInterfacePolicy(cfg, netpolicy.InterfaceStatus{Name: "fixture", Addresses: []string{"2001:db8::12"}}, s)
	if cfg["tsl_network_paused"] != true {
		t.Fatal("disabled address family used")
	}
}
