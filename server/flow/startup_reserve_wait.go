package flow

import (
	"math"
	"time"
)

// Extra startup insurance must not become a second long preload. The media
// target and required index have their own existing verification/wait gates.
const StartupReserveExtraWait = 10 * time.Second

type StartupReserveWait struct {
	baseReadyAt time.Time
	expired     bool
}

// Limit admits only whole additional pieces whose predicted cost fits the
// extra-wait allowance. Once the normal head/index gate is verified, that same
// allowance bounds the real wait too. It never lowers the ordinary media target.
// Piece alignment matters: a one-byte extension can require another 16 MiB.
func (w *StartupReserveWait) Limit(plan ReservePlan, offset, pieceLength int64, baseReady bool, now time.Time) ReservePlan {
	plan.RequestedBytes = plan.TargetBytes
	plan.ExtraWaitBudgetMs = StartupReserveExtraWait.Milliseconds()
	if !baseReady {
		w.baseReadyAt = time.Time{}
	} else if w.baseReadyAt.IsZero() || now.Before(w.baseReadyAt) {
		w.baseReadyAt = now
	} else if now.Sub(w.baseReadyAt) >= StartupReserveExtraWait {
		w.expired = true
	}
	if plan.TargetBytes <= plan.BaseBytes {
		plan.ExtraWaitExpired = w.expired
		return plan
	}
	plan.ExtraWaitExpired = w.expired
	// An unknown rate cannot justify adding a long startup wait. Disabling this
	// optional extension does not disable the normal bitrate-based target.
	if w.expired || pieceLength <= 0 || offset < 0 || plan.BaseBytes <= 0 ||
		!plan.Known || !finitePositive(plan.DeliveryRate) {
		plan.TargetBytes = plan.BaseBytes
	} else {
		offset %= pieceLength
		if plan.TargetBytes-1 > math.MaxInt64-offset {
			plan.TargetBytes = plan.BaseBytes
		} else {
			baseLast := (offset + plan.BaseBytes - 1) / pieceLength
			targetLast := (offset + plan.TargetBytes - 1) / pieceLength
			extra := targetLast - baseLast
			affordable := plan.DeliveryRate * StartupReserveExtraWait.Seconds() / float64(pieceLength)
			if affordable < float64(extra) {
				allowed := int64(math.Max(0, math.Floor(affordable)))
				// The subtraction stays inside the original byte target; clamp
				// to the base for a rounded final piece that crosses its boundary.
				if extra-allowed > math.MaxInt64/pieceLength {
					plan.TargetBytes = plan.BaseBytes
				} else {
					plan.TargetBytes = max(plan.BaseBytes, plan.TargetBytes-(extra-allowed)*pieceLength)
				}
			}
		}
	}
	plan.ExtraWaitLimited = plan.TargetBytes < plan.RequestedBytes
	if plan.ExtraWaitLimited {
		plan.Limited = true
	}
	plan.AdditionalFillSeconds = nil
	if finitePositive(plan.DeliveryRate) && pieceLength > 0 && offset >= 0 &&
		plan.BaseBytes > 0 && plan.TargetBytes-1 <= math.MaxInt64-offset%pieceLength {
		first := (offset%pieceLength + plan.BaseBytes - 1) / pieceLength
		last := (offset%pieceLength + plan.TargetBytes - 1) / pieceLength
		seconds := float64(last-first) * float64(pieceLength) / plan.DeliveryRate
		plan.AdditionalFillSeconds = &seconds
	}
	return plan
}
