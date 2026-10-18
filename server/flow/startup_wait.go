package flow

import "time"

const StartupWaitBase = 2 * time.Minute
const StartupWaitMaximum = 5 * time.Minute

type startupWaitPoint struct {
	at    time.Time
	bytes int64
}

// StartupWait renews a slow Adaptive startup only while verified bootstrap data
// exists and useful gate bytes are still arriving. It cannot bypass an index or
// piece-verification gate. Ordinary profiles keep the two-minute limit.
type StartupWait struct {
	started, deadline, lastProgress time.Time
	adaptive                        bool
	maximumBytes                    int64
	points                          []startupWaitPoint
}

func NewStartupWait(started time.Time, adaptive bool) *StartupWait {
	return &StartupWait{started: started, deadline: started.Add(StartupWaitBase), adaptive: adaptive}
}

func (w *StartupWait) Budget() time.Duration { return w.deadline.Sub(w.started) }

func (w *StartupWait) Expired(now time.Time, received int64, bootstrapVerified bool) bool {
	if received > w.maximumBytes {
		w.maximumBytes = received
		w.lastProgress = now
	}
	// At most eight five-second observations, independent of the preload tick.
	if len(w.points) == 0 || now.Sub(w.points[len(w.points)-1].at) >= 5*time.Second {
		w.points = append(w.points, startupWaitPoint{now, w.maximumBytes})
		for len(w.points) > 1 && now.Sub(w.points[1].at) >= 30*time.Second {
			w.points = w.points[1:]
		}
		if len(w.points) > 8 {
			w.points = w.points[len(w.points)-8:]
		}
	}
	if now.Before(w.deadline) {
		return false
	}
	hard := w.started.Add(StartupWaitMaximum)
	if !w.adaptive || !now.Before(hard) || !bootstrapVerified || now.Sub(w.lastProgress) > 10*time.Second || len(w.points) < 3 {
		return true
	}
	old := w.points[0]
	if now.Sub(old.at) < 10*time.Second || w.maximumBytes-old.bytes < 1<<20 {
		return true
	}
	w.deadline = now.Add(30 * time.Second)
	if w.deadline.After(hard) {
		w.deadline = hard
	}
	return false
}
