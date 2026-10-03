package torr

import (
	"server/flow"
	"server/lt"
	"time"
)

// One writer per session, joined before native destruction. No disk work runs
// on a playback goroutine. Existing native address-change/reannounce recovery
// remains authoritative; persisted nodes are bootstrap hints, not connectivity.
func (bt *BTServer) saveDHTLifecycle(s *lt.Session, stop <-chan struct{}, done chan<- struct{}, enabled bool, name string, history *flow.History) {
	defer close(done)
	if !enabled {
		return
	}
	save := func() {
		data, err := s.DHTState()
		if err == nil {
			nodes, countErr := lt.DHTStateNodes(data)
			if countErr != nil {
				err = countErr
			} else if nodes == 0 {
				return
			}
		}
		if err == nil {
			err = flow.WriteDHTFile(name, data)
		}
		if err == nil {
			history.Record(flow.HistoryEvent{Type: "dht", Stage: "DHT_SAVED", Bytes: int64(len(data))})
		} else {
			history.Record(flow.HistoryEvent{Type: "dht", Stage: "DHT_IGNORED", Code: 1})
		}
	}
	tick := time.NewTicker(5 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			save()
			return
		case <-tick.C:
			save()
		}
	}
}
