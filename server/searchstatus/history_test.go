package searchstatus

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestEndpointHistoryBoundsPrivacyAndCredentialPartition(t *testing.T) {
	key := EndpointKey("torznab", "https://example.invalid/?apikey=secret", "secret")
	for i := 0; i < 30; i++ {
		Record(key, Status{Source: "torznab", State: "ok", QueryMode: "generic", ElapsedMS: int64(i)})
	}
	row := History(key)
	if len(row.Attempts) != 20 || row.Attempts[0].ElapsedMS != 10 || row.LastSuccess == nil {
		t.Fatal(row)
	}
	raw, _ := json.Marshal(row)
	if strings.Contains(string(raw), "secret") || strings.Contains(key, "example") {
		t.Fatal("credential disclosure")
	}
	if len(History(EndpointKey("torznab", "https://example.invalid/?apikey=secret", "changed")).Attempts) != 0 {
		t.Fatal("credentials reused history")
	}
	row.Attempts[0].State = "mutated"
	if History(key).Attempts[0].State != "ok" {
		t.Fatal("mutable shared history")
	}
	for i := 0; i < 140; i++ {
		Record(fmt.Sprint("fixture", i), Status{Source: "torznab", State: "timeout"})
		SetCooldown(fmt.Sprint("fixture", i), time.Now().Add(time.Second))
	}
	histories.Lock()
	defer histories.Unlock()
	if len(histories.rows) > 128 {
		t.Fatal("unbounded endpoints")
	}
}
