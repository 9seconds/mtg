package logthrottle

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestThrottleAllowsOnePerIntervalAndCountsTheRest(t *testing.T) {
	t.Parallel()

	now := time.Unix(1_000_000, 0)

	throttle := New(10 * time.Second)
	throttle.now = func() time.Time { return now }

	if ok, suppressed := throttle.Allow(); !ok || suppressed != 0 {
		t.Fatalf("the first message must pass: ok=%v suppressed=%d", ok, suppressed)
	}

	for range 5 {
		now = now.Add(time.Second)

		if ok, _ := throttle.Allow(); ok {
			t.Fatal("a message within the interval must be suppressed")
		}
	}

	now = now.Add(5 * time.Second)

	if ok, suppressed := throttle.Allow(); !ok || suppressed != 5 {
		t.Fatalf("after the interval: want ok with 5 suppressed, got ok=%v suppressed=%d", ok, suppressed)
	}

	now = now.Add(10 * time.Second)

	if ok, suppressed := throttle.Allow(); !ok || suppressed != 0 {
		t.Fatalf("the counter must reset: ok=%v suppressed=%d", ok, suppressed)
	}
}

func TestThrottleConcurrentLetsOneThrough(t *testing.T) {
	t.Parallel()

	throttle := New(time.Hour)

	var (
		allowed atomic.Int32
		wg      sync.WaitGroup
	)

	for range 100 {
		wg.Go(func() {
			if ok, _ := throttle.Allow(); ok {
				allowed.Add(1)
			}
		})
	}

	wg.Wait()

	if allowed.Load() != 1 {
		t.Fatalf("exactly one message must pass, got %d", allowed.Load())
	}
}
