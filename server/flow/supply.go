package flow

// BufferSupply describes delivery at the server's current demand position. It
// does not estimate the player's internal buffer, decoder state or Wi-Fi speed.
type BufferSupply struct {
	Known        bool     `json:"known"`
	State        string   `json:"state"`
	DemandRate   float64  `json:"demand_rate"`
	DeliveryRate float64  `json:"delivery_rate"`
	DeficitRate  float64  `json:"deficit_rate"`
	Runway       *float64 `json:"runway_seconds,omitempty"`
}

func AssessBufferSupply(demand float64, buffer int64, supply DeliveryEvidence) BufferSupply {
	s := BufferSupply{State: "UNKNOWN"}
	if !finitePositive(demand) || !qualifiedSupply(supply) {
		return s
	}
	s.Known, s.State = true, "SUFFICIENT_SUPPLY"
	s.DemandRate, s.DeliveryRate = demand, supply.LongRate
	s.DeficitRate = max(0, demand-supply.LongRate)
	if s.DeficitRate > 0 {
		s.State = "SUPPLY_DEFICIT"
		if seconds, known := BufferExhaustionSeconds(max(0, buffer), demand, supply.LongRate); known {
			s.Runway = &seconds
		}
	} else if supply.ShortSamples >= 3 && supply.ShortRate < demand {
		s.State = "RECENT_GAPS"
	}
	return s
}
