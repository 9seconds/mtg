//go:build !linux

package network

import "syscall"

// ListenerMSSSupported tells whether the platform can set the MSS of a
// listening socket (TCP_MAXSEG). Outside Linux network.client-mss is ignored.
const ListenerMSSSupported = false

// ListenControlMSS does nothing outside Linux: client-mss works only on
// Linux.
func ListenControlMSS(_ int) func(network, address string, conn syscall.RawConn) error {
	return func(_, _ string, _ syscall.RawConn) error {
		return nil
	}
}
