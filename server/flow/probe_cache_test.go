package flow

import (
	"fmt"
	"testing"
	"time"
)

func TestProbeCacheIdentityRetryAndBound(t *testing.T) {
	var c ProbeCache
	now := time.Now()
	key := ProbeKey{"hash", 1, 100, "episode.mkv"}
	lease := c.Begin(key, now)
	if lease == 0 || c.Begin(key, now) != 0 {
		t.Fatal("probe deduplication")
	}
	c.Finish(key, lease, ProbeResult{}, false, now)
	if c.Begin(key, now.Add(59*time.Second)) != 0 {
		t.Fatal("failure backoff")
	}
	lease = c.Begin(key, now.Add(time.Minute))
	if lease == 0 {
		t.Fatal("failure retry missing")
	}
	c.Finish(key, lease, ProbeResult{"800000", 60}, true, now.Add(time.Minute))
	if r, ok := c.Get(key, now.Add(time.Hour)); !ok || r.Duration != 60 {
		t.Fatal("success missing")
	}
	changed := key
	changed.Size++
	if _, ok := c.Get(changed, now); ok {
		t.Fatal("wrong file identity")
	}
	if _, ok := c.Get(key, now.Add(25*time.Hour)); ok {
		t.Fatal("expired result")
	}
	for i := 0; i < 200; i++ {
		c.Begin(ProbeKey{fmt.Sprint(i), 1, 100, "file"}, now.Add(time.Duration(i)*time.Second))
	}
	if len(c.entries) != 128 {
		t.Fatal("unbounded cache", len(c.entries))
	}
}

func TestExpiredProbeCannotOverwriteReplacement(t *testing.T) {
	var cache ProbeCache
	now := time.Now()
	key := ProbeKey{"hash", 1, 100, "episode.mkv"}
	old := cache.Begin(key, now)
	replacement := cache.Begin(key, now.Add(36*time.Second))
	if old == 0 || replacement == 0 || old == replacement {
		t.Fatal("lease not replaced")
	}
	cache.Finish(key, old, ProbeResult{"1", 1}, true, now.Add(37*time.Second))
	if _, ok := cache.Get(key, now.Add(38*time.Second)); ok {
		t.Fatal("stale result replaced active lease")
	}
	cache.Finish(key, replacement, ProbeResult{"800000", 60}, true, now.Add(39*time.Second))
	if result, ok := cache.Get(key, now.Add(40*time.Second)); !ok || result.Duration != 60 {
		t.Fatal("replacement result lost")
	}
}
