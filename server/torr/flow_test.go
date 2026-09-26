package torr

import (
	"testing"
	"time"

	"server/settings"
	"server/torr/state"
)

func TestWarmTimeoutUsesFlowSetting(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	f := settings.DefaultFlowSettings()
	f.WarmSessionTimeoutSec = 900
	settings.StoreBTsets(&settings.BTSets{TorrentDisconnectTimeout: 30, Flow: f})
	if got := torrentExpireTimeout(); got != 900*time.Second {
		t.Fatalf("Flow timeout = %v", got)
	}
	f.Enabled = false
	if got := torrentExpireTimeout(); got != 30*time.Second {
		t.Fatalf("upstream timeout = %v", got)
	}
}

func TestWarmExpiryIgnoresStatusPollExtension(t *testing.T) {
	old := settings.BTsets()
	t.Cleanup(func() { settings.StoreBTsets(old) })
	settings.StoreBTsets(&settings.BTSets{Flow: settings.DefaultFlowSettings()})
	now := time.Now()
	tor := &Torrent{Stat: state.TorrentWorking, warmIdleSince: now.Add(-601 * time.Second), expiredTime: now.Add(time.Hour)}
	if !tor.expired(now) {
		t.Fatal("warm session should expire despite a later status-poll deadline")
	}
	tor.warmIdleSince = now.Add(-599 * time.Second)
	if tor.expired(now) {
		t.Fatal("warm session expired too soon")
	}
}
