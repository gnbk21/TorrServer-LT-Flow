package lt

import (
	"bytes"
	"encoding/binary"
	"net"
	"strconv"
	"testing"
	"time"
)

// Empty routing state, an immediate magnet lookup and a bootstrap node that is
// initially unavailable exercise the cold-start ordering behind upstream 147.1.
// There are no public DHT nodes, trackers or real torrent identities in this test.
func TestDHTColdMagnetFindsPeersAfterDelayedBootstrap(t *testing.T) {
	var local net.IP
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, _ := iface.Addrs()
		for _, address := range addresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				local = ip.To4()
				break
			}
		}
		if local != nil {
			break
		}
	}
	if local == nil {
		t.Fatal("cold DHT fixture needs a local IPv4 interface")
	}
	node, err := net.ListenUDP("udp4", &net.UDPAddr{IP: local})
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()
	// Reserve a local destination; no real peer or payload is involved.
	peer, err := net.ListenTCP("tcp4", &net.TCPAddr{IP: local})
	if err != nil {
		t.Fatal(err)
	}
	defer peer.Close()
	s, err := NewSession(SessionConfig{
		"enable_dht": true, "enable_lsd": false, "enable_upnp": false, "enable_natpmp": false,
		"dht_bootstrap_nodes": node.LocalAddr().String(), "dht_ignore_dark_internet": false,
		"dht_enforce_node_id": false, "dht_restrict_routing_ips": false, "dht_restrict_search_ips": false,
		"listen_interfaces": net.JoinHostPort(local.String(), "0"), "dht_announce_interval": 60,
		"alert_mask": 0x7fffffff,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	initial, err := s.DHTState()
	if err != nil {
		t.Fatal(err)
	}
	if count, err := DHTStateNodes(initial); err != nil || count != 0 {
		t.Fatalf("cold fixture already has routing nodes: %d %v", count, err)
	}
	h, err := s.AddTorrent(AddTorrentParams{Link: "magnet:?xt=urn:btih:1111111111111111111111111111111111111111", SavePath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	if err := h.ForceDhtAnnounce(); err != nil {
		t.Fatal(err)
	}
	var available time.Time
	deadline := started.Add(35 * time.Second)
	buf := make([]byte, 4096)
	var firstContact, firstLookup time.Time
	for time.Now().Before(deadline) {
		alerts, err := s.PopAlerts()
		if err != nil {
			t.Fatal(err)
		}
		for _, alert := range alerts {
			if alert.Type == "dht_reply" && alert.Peers > 0 {
				if firstLookup.IsZero() {
					t.Fatal("peer result without fixture lookup")
				}
				t.Logf("cold DHT: bootstrap available=%s first_contact=%s first_get_peers=%s peer_result=%s", available.Sub(started), firstContact.Sub(started), firstLookup.Sub(started), time.Since(started))
				return
			}
		}
		node.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
		n, from, err := node.ReadFromUDP(buf)
		if err != nil {
			if e, ok := err.(net.Error); ok && e.Timeout() {
				continue
			}
			t.Fatal(err)
		}
		if firstContact.IsZero() {
			firstContact = time.Now()
			// Delay from the actual first query, rather than from session
			// creation: even a slow runner must exercise an unavailable node.
			available = firstContact.Add(time.Second)
		}
		if time.Now().Before(available) {
			continue
		}
		packet := buf[:n]
		at := bytes.Index(packet, []byte("1:t"))
		if at < 0 {
			continue
		}
		at += 3
		colon := bytes.IndexByte(packet[at:], ':')
		if colon < 1 {
			t.Fatal("fixture received malformed transaction")
		}
		length, err := strconv.Atoi(string(packet[at : at+colon]))
		if err != nil || length < 1 || length > 16 || at+colon+1+length > n {
			t.Fatal("invalid transaction ID")
		}
		tx := packet[at+colon+1 : at+colon+1+length]
		reply := append([]byte("d1:rd2:id20:"), bytes.Repeat([]byte{0x31}, 20)...)
		if bytes.Contains(packet, []byte("1:q9:get_peers")) {
			if firstLookup.IsZero() {
				firstLookup = time.Now()
			}
			compact := append([]byte(nil), local...)
			compact = binary.BigEndian.AppendUint16(compact, uint16(peer.Addr().(*net.TCPAddr).Port))
			reply = append(reply, []byte("5:token2:ok6:valuesl6:")...)
			reply = append(reply, compact...)
			reply = append(reply, 'e')
		} else {
			reply = append(reply, []byte("5:nodes0:")...)
		}
		reply = append(reply, []byte("e1:t"+strconv.Itoa(len(tx))+":")...)
		reply = append(reply, tx...)
		reply = append(reply, []byte("1:y1:re")...)
		if _, err := node.WriteToUDP(reply, from); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("cold DHT did not return fixture peers within 35s: contacted=%t lookup=%t", !firstContact.IsZero(), !firstLookup.IsZero())
}

func dhtFixture(endpoint []byte) []byte {
	return append(append([]byte("d9:dht stated5:nodesl6:"), endpoint...), []byte("ee8:settingsd16:connection_speedi999eee")...)
}

func TestDHTStateNativeRoundTripAndIsolation(t *testing.T) {
	endpoint := []byte{127, 0, 0, 1, 0x1a, 0xe1}
	clean, err := NormalizeDHTState(dhtFixture(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(clean, []byte("settings")) || !bytes.Contains(clean, endpoint) {
		t.Fatalf("non-DHT state retained or routing node lost: %q", clean)
	}
	again, err := NormalizeDHTState(clean)
	if err != nil || !bytes.Equal(clean, again) {
		t.Fatalf("round trip: %v", err)
	}
	s, err := NewSessionWithDHT(SessionConfig{"enable_dht": false, "enable_upnp": false, "enable_natpmp": false, "listen_interfaces": "127.0.0.1:0", "connection_speed": 37}, clean)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.SettingInt("connection_speed")
	if err != nil || got != 37 {
		t.Fatalf("persisted state changed configured settings: %d %v", got, err)
	}
	for _, bad := range [][]byte{nil, []byte("garbage"), []byte("de"), append(clean, 'x'), []byte("d9:dht stated5:nodesli1eeee"), make([]byte, 1<<20+1)} {
		if _, err := NormalizeDHTState(bad); err == nil {
			t.Fatalf("accepted invalid state of %d bytes", len(bad))
		}
	}
}

func TestDHTRestoredNodeIsContactedWithoutPublicBootstrap(t *testing.T) {
	// Libtorrent excludes loopback-only DHT sockets. Use a local interface,
	// with no public bootstrap nodes and no dependency on Internet access.
	var local net.IP
	interfaces, err := net.Interfaces()
	if err != nil {
		t.Fatal(err)
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addresses, _ := iface.Addrs()
		for _, address := range addresses {
			ip, _, parseErr := net.ParseCIDR(address.String())
			if parseErr == nil && ip.To4() != nil && !ip.IsLoopback() && !ip.IsLinkLocalUnicast() {
				local = ip.To4()
				break
			}
		}
		if local != nil {
			break
		}
	}
	if local == nil {
		t.Fatal("DHT fixture requires a local IPv4 interface")
	}
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: local})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	port := conn.LocalAddr().(*net.UDPAddr).Port
	endpoint := append(append([]byte(nil), local...), byte(port>>8), byte(port))
	state, err := NormalizeDHTState(dhtFixture(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSessionWithDHT(SessionConfig{"enable_dht": true, "enable_upnp": false, "enable_natpmp": false, "dht_bootstrap_nodes": "", "dht_ignore_dark_internet": false, "listen_interfaces": net.JoinHostPort(local.String(), "0"), "alert_mask": 0x7fffffff}, state)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	deadline := time.Now().Add(10 * time.Second)
	began := time.Now()
	buf := make([]byte, 2048)
	var n int
	var sender *net.UDPAddr
	var trace []Alert
	for time.Now().Before(deadline) {
		alerts, _ := s.PopAlerts() // production's existing alert pump drives readiness
		trace = append(trace, alerts...)
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, sender, err = conn.ReadFromUDP(buf)
		if err == nil {
			break
		}
	}
	if err != nil {
		for _, alert := range trace {
			if alert.Type == "dht_log" || alert.Type == "listen_failed" || alert.Type == "listen_succeeded" {
				t.Log(alert.Type, alert.Message)
			}
		}
		t.Fatalf("restored DHT node on port %s was not contacted: %v", strconv.Itoa(port), err)
	}
	t.Logf("restored local DHT node contacted in %s, without public bootstrap", time.Since(began))
	if !bytes.Contains(buf[:n], []byte("1:q")) {
		t.Fatal("not a DHT query")
	}
	// Answer the native query so the snapshot contains a verified routing node,
	// rather than merely accepting a syntactically valid empty state.
	i := bytes.Index(buf[:n], []byte("1:t")) + 3
	if i < 3 {
		t.Fatal("missing transaction ID")
	}
	colon := bytes.IndexByte(buf[i:n], ':')
	if colon < 1 {
		t.Fatal("invalid transaction length")
	}
	length, parseErr := strconv.Atoi(string(buf[i : i+colon]))
	if parseErr != nil || length < 1 || length > 8 || i+colon+1+length > n {
		t.Fatal("invalid transaction ID")
	}
	tx := buf[i+colon+1 : i+colon+1+length]
	reply := append([]byte("d1:rd2:id20:"), bytes.Repeat([]byte{1}, 20)...)
	reply = append(reply, []byte("5:nodes0:e1:t"+strconv.Itoa(length)+":")...)
	reply = append(reply, tx...)
	reply = append(reply, []byte("1:y1:re")...)
	if _, err := conn.WriteToUDP(reply, sender); err != nil {
		t.Fatal(err)
	}
	routingDeadline := time.Now().Add(3 * time.Second)
	for {
		s.PopAlerts()
		data, snapshotErr := s.DHTState()
		nodes, countErr := DHTStateNodes(data)
		if snapshotErr == nil && countErr == nil && nodes > 0 {
			break
		}
		if time.Now().After(routingDeadline) {
			t.Fatalf("verified routing node was not saved: %v %v", snapshotErr, countErr)
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Rebind the live session without rebuilding it. Restored/verified hints
	// must also bootstrap the new UDP socket after a network/listener change.
	// A free UDP port can still belong to an existing TCP listener. Libtorrent
	// binds both protocols and may then choose another port, which would make
	// this fixture's exact-source-port assertion fail for the wrong reason.
	var newPort int
	for attempt := 0; attempt < 8; attempt++ {
		tcpReservation, reserveErr := net.ListenTCP("tcp4", &net.TCPAddr{IP: local})
		if reserveErr != nil {
			t.Fatal(reserveErr)
		}
		port := tcpReservation.Addr().(*net.TCPAddr).Port
		udpReservation, reserveErr := net.ListenUDP("udp4", &net.UDPAddr{IP: local, Port: port})
		if reserveErr != nil {
			tcpReservation.Close()
			continue
		}
		newPort = port
		udpReservation.Close()
		tcpReservation.Close()
		break
	}
	if newPort == 0 {
		t.Fatal("could not reserve a matching TCP/UDP fixture port")
	}
	if err := s.ApplySettings(SessionConfig{"listen_interfaces": net.JoinHostPort(local.String(), strconv.Itoa(newPort))}); err != nil {
		t.Fatal(err)
	}
	recoveryDeadline := time.Now().Add(5 * time.Second)
	recovered := false
	alertCounts := make(map[string]int)
	queryPorts := make(map[int]int)
	newListenerReady := false
	for time.Now().Before(recoveryDeadline) {
		alerts, popErr := s.PopAlerts()
		if popErr != nil {
			t.Fatal(popErr)
		}
		for _, alert := range alerts {
			alertCounts[alert.Type]++
			if alert.Type == "listen_succeeded" && alert.Port == newPort && alert.Transport == "udp" {
				newListenerReady = true
			}
		}
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		n, sender, err = conn.ReadFromUDP(buf)
		if err == nil && bytes.Contains(buf[:n], []byte("1:q")) {
			queryPorts[sender.Port]++
		}
		if err == nil && sender.Port == newPort && bytes.Contains(buf[:n], []byte("1:q")) {
			recovered = true
			break
		}
	}
	if !recovered {
		t.Fatalf("new UDP listener did not reuse verified DHT hints: ready=%t new_port=%d query_ports=%v alert_counts=%v", newListenerReady, newPort, queryPorts, alertCounts)
	}
	if data, err := s.DHTState(); err != nil || len(data) == 0 {
		t.Fatalf("native snapshot: %v", err)
	}
}
