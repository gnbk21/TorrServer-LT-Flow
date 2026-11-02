package searchstatus

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"time"
)

// Fingerprints partition changed endpoints/credentials without retaining either.
func EndpointKey(source, host, key string) string {
	sum := sha256.Sum256([]byte(source + "\x00" + host + "\x00" + key))
	return hex.EncodeToString(sum[:])
}

type Observation struct {
	Status
	At time.Time `json:"at"`
}
type EndpointHistory struct {
	LastSuccess   *time.Time    `json:"last_success,omitempty"`
	CooldownUntil *time.Time    `json:"cooldown_until,omitempty"`
	Attempts      []Observation `json:"attempts"`
}

var histories = struct {
	sync.Mutex
	rows map[string]EndpointHistory
}{rows: map[string]EndpointHistory{}}

func Record(key string, status Status) {
	recordMetric(status)
	histories.Lock()
	defer histories.Unlock()
	if len(histories.rows) >= 128 {
		if _, ok := histories.rows[key]; !ok {
			for k := range histories.rows {
				delete(histories.rows, k)
				break
			}
		}
	}
	row := histories.rows[key]
	now := time.Now().UTC()
	if status.State == "ok" || status.State == "empty" {
		row.LastSuccess = &now
	}
	row.Attempts = append(row.Attempts, Observation{Status: status, At: now})
	if len(row.Attempts) > 20 {
		row.Attempts = append([]Observation(nil), row.Attempts[len(row.Attempts)-20:]...)
	}
	histories.rows[key] = row
}
func History(key string) EndpointHistory {
	histories.Lock()
	defer histories.Unlock()
	row := histories.rows[key]
	row.Attempts = append([]Observation{}, row.Attempts...)
	if row.LastSuccess != nil {
		at := *row.LastSuccess
		row.LastSuccess = &at
	}
	if row.CooldownUntil != nil {
		at := *row.CooldownUntil
		row.CooldownUntil = &at
	}
	return row
}
func SetCooldown(key string, until time.Time) {
	histories.Lock()
	defer histories.Unlock()
	if len(histories.rows) >= 128 {
		if _, exists := histories.rows[key]; !exists {
			return
		}
	}
	row := histories.rows[key]
	row.CooldownUntil = &until
	histories.rows[key] = row
}
