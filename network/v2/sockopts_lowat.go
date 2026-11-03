//go:build linux || darwin

package network

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// setNotSentLowat sets TCP_NOTSENT_LOWAT to value bytes. The option
// limits the amount of unsent data queued in the kernel write buffer:
// once the unsent backlog drops below the threshold, the socket becomes
// writable again, applying back-pressure to the relay loop instead of
// piling up data in kernel buffers. This reduces per-connection memory
// and bufferbloat.
//
// A non-positive value disables the call, leaving the kernel default in
// effect (no upper bound beyond SO_SNDBUF).
func setNotSentLowat(conn syscall.RawConn, value int) {
	if value <= 0 {
		return
	}

	conn.Control(func(fd uintptr) { //nolint: errcheck
		unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT, value) //nolint: errcheck
	})
}
