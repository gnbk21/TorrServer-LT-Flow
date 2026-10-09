package torr

import (
	"net"
	"server/lt"
	"server/netpolicy"
	"server/settings"
	"strconv"
	"strings"
)

func applyInterfacePolicy(cfg lt.SessionConfig, policy netpolicy.InterfaceStatus, s *settings.BTSets) {
	if policy.Name == "" {
		return
	}
	var listen, outgoing []string
	for _, address := range policy.Addresses {
		ip := net.ParseIP(address)
		if ip == nil || (ip.To4() == nil && !s.EnableIPv6) {
			continue
		}
		listen = append(listen, net.JoinHostPort(address, strconv.Itoa(s.PeersListenPort)))
		outgoing = append(outgoing, address)
	}
	paused := len(outgoing) == 0
	if paused {
		listen = []string{"127.0.0.1:0"}
		outgoing = []string{"127.0.0.1"}
	}
	cfg["listen_interfaces"] = strings.Join(listen, ",")
	cfg["outgoing_interfaces"] = strings.Join(outgoing, ",")
	cfg["tsl_network_paused"] = paused
	// Port mapping and multicast discovery belong to the physical LAN. Avoid
	// announcing a selected VPN binding through another adapter.
	cfg["enable_upnp"] = false
	cfg["enable_natpmp"] = false
	cfg["enable_lsd"] = false
	if paused {
		cfg["enable_dht"] = false
		cfg["enable_outgoing_tcp"] = false
		cfg["enable_outgoing_utp"] = false
		cfg["enable_incoming_tcp"] = false
		cfg["enable_incoming_utp"] = false
	}
}

func (bt *BTServer) reconcileInterfacePolicy(fingerprint *string) error {
	policy, _ := netpolicy.Refresh()
	key := policy.Name + ":" + policy.State + ":" + strings.Join(policy.Addresses, ",")
	if policy.Name == "" || key == *fingerprint {
		return nil
	}
	all, err := buildSessionConfig()
	if err != nil {
		return err
	}
	cfg := lt.SessionConfig{}
	for _, key := range []string{"listen_interfaces", "outgoing_interfaces", "tsl_network_paused", "enable_upnp", "enable_natpmp", "enable_lsd", "enable_dht", "enable_outgoing_tcp", "enable_outgoing_utp", "enable_incoming_tcp", "enable_incoming_utp"} {
		if value, ok := all[key]; ok {
			cfg[key] = value
		}
	}
	session := bt.Session()
	if session == nil {
		return nil
	}
	if err = session.ApplySettings(cfg); err != nil {
		return err
	}
	*fingerprint = key
	return nil
}
