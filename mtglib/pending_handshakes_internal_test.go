package mtglib

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
)

var (
	testIPA = net.ParseIP("198.51.100.1")
	testIPB = net.ParseIP("198.51.100.2")
)

func TestPendingHandshakesDisabled(t *testing.T) {
	t.Parallel()

	p := newPendingHandshakes()

	for range 100 {
		release, action, ok := p.acquire(testIPA, 0, false)
		if !ok || release != nil || action != "" {
			t.Fatal("limit 0 must admit without tracking")
		}
	}

	if p.trackedIPs() != 0 {
		t.Fatal("nothing must be tracked with limit 0")
	}
}

func TestPendingHandshakesRejectOverLimitAndRelease(t *testing.T) {
	t.Parallel()

	p := newPendingHandshakes()

	var releases []func()

	for range 3 {
		release, action, ok := p.acquire(testIPA, 3, false)
		if !ok || action != "" {
			t.Fatal("expected admission under the limit")
		}

		releases = append(releases, release)
	}

	if _, action, ok := p.acquire(testIPA, 3, false); ok || action != PendingHandshakeRejected {
		t.Fatal("expected rejection over the limit")
	}

	if p.pendingFor(testIPA) != 3 {
		t.Fatalf("a rejection must not take a slot, pending %d", p.pendingFor(testIPA))
	}

	releases[0]()
	releases[0]() // release is idempotent

	if p.pendingFor(testIPA) != 2 {
		t.Fatalf("expected 2 pending after one release, got %d", p.pendingFor(testIPA))
	}

	if _, _, ok := p.acquire(testIPA, 3, false); !ok {
		t.Fatal("a released slot must be available again")
	}
}

func TestPendingHandshakesAddressesAreIndependent(t *testing.T) {
	t.Parallel()

	p := newPendingHandshakes()

	if _, _, ok := p.acquire(testIPA, 1, false); !ok {
		t.Fatal("first address must be admitted")
	}

	if _, _, ok := p.acquire(testIPA, 1, false); ok {
		t.Fatal("first address must be over the limit")
	}

	if _, _, ok := p.acquire(testIPB, 1, false); !ok {
		t.Fatal("another address must not be affected")
	}

	// IPv4-mapped IPv6 is the same address.
	if _, _, ok := p.acquire(net.ParseIP("::ffff:198.51.100.1"), 1, false); ok {
		t.Fatal("IPv4-mapped form must count as the same address")
	}
}

func TestPendingHandshakesDryRun(t *testing.T) {
	t.Parallel()

	p := newPendingHandshakes()

	r1, action, ok := p.acquire(testIPA, 1, true)
	if !ok || action != "" {
		t.Fatal("first must be admitted silently")
	}

	r2, action, ok := p.acquire(testIPA, 1, true)
	if !ok || action != PendingHandshakeObserved {
		t.Fatalf("over the limit in dry run must be admitted and observed, got %q %v", action, ok)
	}

	if p.pendingFor(testIPA) != 2 {
		t.Fatal("dry run must keep counting")
	}

	r1()
	r2()

	if p.trackedIPs() != 0 {
		t.Fatal("entries must be removed when idle")
	}
}

func TestPendingHandshakesConcurrentNeverExceedsLimit(t *testing.T) {
	t.Parallel()

	p := newPendingHandshakes()

	const limit = 8

	var peak atomic.Uint32

	var wg sync.WaitGroup

	for range 16 {
		wg.Go(func() {
			for range 2000 {
				release, _, ok := p.acquire(testIPA, limit, false)
				if !ok {
					continue
				}

				now := p.pendingFor(testIPA)

				for {
					old := peak.Load()
					if now <= old || peak.CompareAndSwap(old, now) {
						break
					}
				}

				release()
			}
		})
	}

	wg.Wait()

	if peak.Load() > limit {
		t.Fatalf("peak %d exceeds limit %d", peak.Load(), limit)
	}

	if p.trackedIPs() != 0 {
		t.Fatal("entries must be removed when idle")
	}
}
