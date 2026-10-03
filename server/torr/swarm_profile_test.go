package torr

import (
	"testing"

	"server/lt"
	"server/settings"
)

func TestFlowSwarmProfilesPreserveCompatibilityAndUserSettings(t *testing.T) {
	base := func() lt.SessionConfig {
		return lt.SessionConfig{
			"connection_speed": 250, "torrent_connect_boost": 100,
			"peer_connect_timeout": 7, "strict_end_game_mode": true,
			"active_limit": -1, "proxy_type": 2,
			"close_redundant_connections": false,
		}
	}
	legacy := base()
	applyFlowSwarmProfile(legacy, settings.DefaultFlowSettings(), false)
	if legacy["connection_speed"] != 250 || legacy["peer_connect_timeout"] != 7 {
		t.Fatalf("legacy values changed: %v", legacy)
	}
	for _, tc := range []struct {
		profile string
		speed   any
		boost   any
	}{
		{"conservative", nil, nil},
		{"balanced", 50, 50},
		{"aggressive", 100, 80},
	} {
		cfg := base()
		f := settings.DefaultFlowSettings()
		f.SwarmProfile = tc.profile
		applyFlowSwarmProfile(cfg, f, false)
		if cfg["connection_speed"] != tc.speed || cfg["torrent_connect_boost"] != tc.boost {
			t.Fatalf("%s ramp: %v", tc.profile, cfg)
		}
		if _, ok := cfg["peer_connect_timeout"]; ok {
			t.Fatalf("%s retained shortened peer timeout", tc.profile)
		}
		if cfg["active_limit"] != -1 || cfg["proxy_type"] != 2 || cfg["close_redundant_connections"] != false {
			t.Fatalf("%s lost unrelated settings", tc.profile)
		}
	}
	cfg := base()
	f := settings.DefaultFlowSettings()
	f.SwarmProfile = "custom"
	f.SwarmCustom = settings.FlowSwarmCustom{ConnectionSpeed: 75, PeerConnectTimeout: 20}
	applyFlowSwarmProfile(cfg, f, true)
	if cfg["connection_speed"] != 75 || cfg["peer_connect_timeout"] != 20 || cfg["strict_end_game_mode"] != false {
		t.Fatalf("custom overrides or endgame toggle lost: %v", cfg)
	}
	if _, ok := cfg["torrent_connect_boost"]; ok {
		t.Fatalf("unset custom override should use libtorrent default: %v", cfg)
	}
	if cfg["close_redundant_connections"] != false {
		t.Fatal("custom profile discarded lazy/warm streaming peer retention")
	}
}
