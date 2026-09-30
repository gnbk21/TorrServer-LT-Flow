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
	if !c.Begin(key, now) || c.Begin(key, now) {
		t.Fatal("probe deduplication")
	}
	c.Finish(key, ProbeResult{}, false, now)
	if c.Begin(key, now.Add(59*time.Second)) || !c.Begin(key, now.Add(time.Minute)) {
		t.Fatal("failure backoff")
	}
	c.Finish(key, ProbeResult{"800000", 60}, true, now.Add(time.Minute))
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
