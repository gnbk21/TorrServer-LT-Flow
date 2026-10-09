package torr

import (
	"server/lt"
	"server/settings"
)

// The fork's existing tuning remains the legacy compatibility profile until
// comparative tests justify selecting a new default. The named profiles clear
// those overrides first, then apply only their intentional differences.
var legacySwarmKeys = []string{
	"strict_end_game_mode", "prioritize_partial_pieces",
	"announce_to_all_tiers", "announce_to_all_trackers",
	"min_announce_interval", "max_suggest_pieces",
	"request_queue_time", "max_out_request_queue", "max_allowed_in_request_queue",
	"send_buffer_watermark", "send_buffer_watermark_factor",
	"connection_speed", "max_peerlist_size", "max_pex_peers",
	"dht_upload_rate_limit", "mixed_mode_algorithm",
	"torrent_connect_boost", "peer_connect_timeout", "piece_timeout",
	"min_reconnect_time", "allow_multiple_connections_per_ip",
	"dht_announce_interval", "auto_scrape_interval", "auto_scrape_min_interval",
	"stop_tracker_timeout", "seed_choking_algorithm",
	"rate_limit_ip_overhead", "send_buffer_low_watermark",
}

func applyFlowSwarmProfile(cfg lt.SessionConfig, f *settings.FlowSettings, disableEndGame bool) {
	if f != nil && f.Enabled && f.CapacityAwareRequests {
		cfg["flow_capacity_aware_requests"] = true
	}
	if f == nil || !f.Enabled || f.SwarmProfile == "legacy" || f.SwarmProfile == "adaptive" {
		return
	}
	for _, key := range legacySwarmKeys {
		delete(cfg, key)
	}
	// Keep the shared streaming retention setting close_redundant_connections
	// false. A lazy torrent has all piece priorities at zero between metadata
	// and playback (and in warm idle), so libtorrent considers it finished and
	// otherwise drops its seeders. The next preload then waits for the normal
	// reconnect backoff even though those peers were healthy. This is a cache
	// lifecycle requirement, like the active_* queue overrides, not swarm ramp
	// tuning. Peer caps and torrent expiry still bound retained connections.
	if disableEndGame {
		cfg["strict_end_game_mode"] = false
	}
	switch f.SwarmProfile {
	case "conservative":
		// The active_* overrides remain to prevent libtorrent's queue manager
		// from pausing a torrent currently being streamed.
	case "balanced":
		cfg["connection_speed"] = 50
		cfg["torrent_connect_boost"] = 50
	case "aggressive":
		cfg["connection_speed"] = 100
		cfg["torrent_connect_boost"] = 80
	case "custom":
		custom := f.SwarmCustom
		for _, entry := range []struct {
			key   string
			value int
		}{
			{"connection_speed", custom.ConnectionSpeed},
			{"torrent_connect_boost", custom.TorrentConnectBoost},
			{"peer_connect_timeout", custom.PeerConnectTimeout},
			{"piece_timeout", custom.PieceTimeout},
			{"request_queue_time", custom.RequestQueueTime},
			{"min_reconnect_time", custom.MinReconnectTime},
		} {
			if entry.value > 0 {
				cfg[entry.key] = entry.value
			}
		}
	}
}
