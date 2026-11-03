package mtglib

import (
	"net"
	"net/netip"
	"sync"
)

// Results of a pending-handshake admission, reported as the action label of
// the pending_handshake_limit metric.
const (
	PendingHandshakeRejected = "rejected"
	PendingHandshakeObserved = "observed"
)

// pendingHandshakes limits concurrent unauthenticated handshakes per client
// IP address.
//
// A connection that never completes the handshake holds a worker until the
// handshake timeout. One address that opens hundreds of such connections can
// keep the proxy busy and starve legitimate clients. Only connections that
// have not proven a secret yet are counted: a slot is released as soon as the
// secret is verified or the connection goes to the fronting domain, so
// authenticated sessions and bursts of short media connections are not
// affected. Entries exist only while an address has pending handshakes.
//
// The limit stops silent and slow connections, not every flood: a connection
// that sends garbage goes to the fronting domain at once, releases its slot
// and then holds a worker under the idle timeout. Addresses are counted as is,
// so IPv6 is limited per /128, and clients behind one shared address (CGNAT)
// share the limit.
type pendingHandshakes struct {
	mu      sync.Mutex
	pending map[netip.Addr]uint32
}

func newPendingHandshakes() *pendingHandshakes {
	return &pendingHandshakes{pending: map[netip.Addr]uint32{}}
}

// acquire takes a slot for ip. limit == 0 disables the check. It returns an
// idempotent release function (nil if nothing was taken), the metric action
// ("" when under the limit) and whether the connection may proceed. In dry-run
// mode an address over the limit is admitted, still tracked, and reported as
// observed.
func (p *pendingHandshakes) acquire(ip net.IP, limit uint32, dryRun bool) (func(), string, bool) {
	if limit == 0 {
		return nil, "", true
	}

	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return nil, "", true
	}

	addr = addr.Unmap()

	p.mu.Lock()

	overLimit := p.pending[addr] >= limit
	if overLimit && !dryRun {
		p.mu.Unlock()

		return nil, PendingHandshakeRejected, false
	}

	p.pending[addr]++
	p.mu.Unlock()

	var once sync.Once

	release := func() {
		once.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()

			if p.pending[addr] <= 1 {
				delete(p.pending, addr)
			} else {
				p.pending[addr]--
			}
		})
	}

	if overLimit {
		return release, PendingHandshakeObserved, true
	}

	return release, "", true
}

// pendingFor returns the number of pending handshakes of ip.
func (p *pendingHandshakes) pendingFor(ip net.IP) uint32 {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return 0
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	return p.pending[addr.Unmap()]
}

// trackedIPs returns the number of addresses with pending handshakes.
func (p *pendingHandshakes) trackedIPs() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	return len(p.pending)
}
