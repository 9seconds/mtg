//go:build linux

package network

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

// ListenerMSSSupported tells whether the platform can set the MSS of a
// listening socket (TCP_MAXSEG). Outside Linux network.client-mss is ignored.
const ListenerMSSSupported = true

// ListenControlMSS returns a Control function for net.ListenConfig that sets
// TCP_MAXSEG on the listening socket before bind/listen. Accepted connections
// inherit the value: the SYN-ACK advertises this MSS to the client, and the
// kernel never sends the client segments larger than that until the
// connection ends (mss_clamp is fixed during the handshake and cannot be
// raised later with setsockopt). mss <= 0 leaves the socket alone.
func ListenControlMSS(mss int) func(network, address string, conn syscall.RawConn) error {
	return func(_, _ string, conn syscall.RawConn) error {
		if mss <= 0 {
			return nil
		}

		var opErr error

		if err := conn.Control(func(fd uintptr) {
			opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG, mss)
		}); err != nil {
			return fmt.Errorf("cannot access listener socket: %w", err)
		}

		if opErr != nil {
			return fmt.Errorf("cannot set TCP_MAXSEG=%d: %w", mss, opErr)
		}

		return nil
	}
}
