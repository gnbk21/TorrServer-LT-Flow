package torrstor

import (
	"server/flow"
	"server/settings"
	"testing"
	"time"
)

func TestNextEpisodeUsesSpareBudgetYieldsAndExpires(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.NextEpisodeWarmup = true
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB, Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(1, mkHash(51), 128, flow.MiB)
	c := s.lookup(1)
	c.SetNextEpisodeWarmup(2, 64*flow.MiB, 60*flow.MiB, true, true, "HEALTHY_BUFFER")
	if demand := c.nextEpisodeDemand(false); len(demand) != 32 {
		t.Fatal(len(demand))
	}
	if c.streamingReserve() != 0 {
		t.Fatal("warmup enlarged a protected reservation")
	}
	if len(c.nextEpisodeDemand(true)) != 0 {
		t.Fatal("blocked playback did not yield")
	}
	c.resourceBudget.Store(16 * flow.MiB)
	if len(c.nextEpisodeDemand(false)) != 0 {
		t.Fatal("warmup exceeded coordinated budget")
	}
	c.resourceBudget.Store(64 * flow.MiB)
	c.backgroundLimited.Store(true)
	if len(c.nextEpisodeDemand(false)) != 0 {
		t.Fatal("pressure did not yield")
	}
	c.backgroundLimited.Store(false)
	c.nextMu.Lock()
	c.nextWarmup.until = time.Now().Add(-time.Second)
	c.nextMu.Unlock()
	if len(c.nextEpisodeDemand(false)) != 0 {
		t.Fatal("abandoned plan stayed alive")
	}
	c.SetNextEpisodeWarmup(2, 64*flow.MiB, 60*flow.MiB, true, true, "HEALTHY_BUFFER")
	f.NextEpisodeWarmup = false
	if len(c.nextEpisodeDemand(false)) != 0 {
		t.Fatal("disabled warmup requested pieces")
	}
	c.ClearNextEpisodeWarmup()
	if c.NextEpisodeWarmupStatus().FileIndex != 0 {
		t.Fatal("cancel retained selection")
	}
}

func TestNextEpisodeSingleGlobalOwnerAndOtherPlaybackWait(t *testing.T) {
	old := settings.BTsets()
	f := settings.DefaultFlowSettings()
	f.NextEpisodeWarmup = true
	settings.StoreBTsets(&settings.BTSets{CacheSize: 64 * flow.MiB, Flow: f})
	t.Cleanup(func() { settings.StoreBTsets(old) })
	s := NewStorage()
	s.callbackOpen(1, mkHash(52), 128, flow.MiB)
	s.callbackOpen(2, mkHash(53), 128, flow.MiB)
	a, b := s.lookup(1), s.lookup(2)
	a.SetNextEpisodeWarmup(2, 64*flow.MiB, 60*flow.MiB, true, true, "HEALTHY_BUFFER")
	b.SetNextEpisodeWarmup(2, 64*flow.MiB, 60*flow.MiB, true, true, "HEALTHY_BUFFER")
	if len(b.nextEpisodeDemand(false)) != 0 || b.NextEpisodeWarmupStatus().Reason != "OTHER_WARMUP" {
		t.Fatal("two warmups requested concurrently")
	}
	b.flowWaiting.Store(1)
	if len(a.nextEpisodeDemand(false)) != 0 {
		t.Fatal("another torrent's read was ignored")
	}
	b.flowWaiting.Store(0)
	a.close()
	b.SetNextEpisodeWarmup(2, 64*flow.MiB, 60*flow.MiB, true, true, "HEALTHY_BUFFER")
	if len(b.nextEpisodeDemand(false)) == 0 {
		t.Fatal("closed owner retained global lease")
	}
}
