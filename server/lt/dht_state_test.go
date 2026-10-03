package lt

import (
	"bytes"
	"net"
	"strconv"
	"testing"
	"time"
)

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
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	port := conn.LocalAddr().(*net.UDPAddr).Port
	endpoint := []byte{127, 0, 0, 1, byte(port >> 8), byte(port)}
	state, err := NormalizeDHTState(dhtFixture(endpoint))
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewSessionWithDHT(SessionConfig{"enable_dht": true, "enable_upnp": false, "enable_natpmp": false, "dht_bootstrap_nodes": "", "dht_ignore_dark_internet": false, "listen_interfaces": "0.0.0.0:0"}, state)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	buf := make([]byte, 2048)
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		t.Fatalf("restored DHT node on port %s was not contacted: %v", strconv.Itoa(port), err)
	}
	if !bytes.Contains(buf[:n], []byte("1:q")) {
		t.Fatal("not a DHT query")
	}
	if data, err := s.DHTState(); err != nil || len(data) == 0 {
		t.Fatalf("native snapshot: %v", err)
	}
}
