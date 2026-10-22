// Package logthrottle limits how often a repetitive log line is written.
//
// Some log lines are written per connection on paths an attacker controls (a
// flood of rejected or broken connections). Writing each of them turns a
// flood into a log flood, while dropping them makes the problem invisible.
// A Throttle lets one line through per interval and counts the rest, so the
// next line can report how many were suppressed.
package logthrottle

import (
	"sync"
	"time"
)

// Throttle allows at most one message per interval. It is safe for
// concurrent use.
type Throttle struct {
	mu         sync.Mutex
	interval   time.Duration
	last       time.Time
	suppressed int
	now        func() time.Time
}

// New returns a Throttle that allows one message per interval.
func New(interval time.Duration) *Throttle {
	return &Throttle{
		interval: interval,
		now:      time.Now,
	}
}

// Allow reports whether a message may be written now. If it may, it also
// returns the number of messages suppressed since the previous allowed one,
// and resets that counter.
func (t *Throttle) Allow() (bool, int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()

	if !t.last.IsZero() && now.Sub(t.last) < t.interval {
		t.suppressed++

		return false, 0
	}

	suppressed := t.suppressed
	t.last = now
	t.suppressed = 0

	return true, suppressed
}
