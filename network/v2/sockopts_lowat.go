//go:build linux || darwin

package network

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// setNotSentLowat sets TCP_NOTSENT_LOWAT which limits the amount of
// unsent data queued in the kernel write buffer. Once unsent data drops
// below this threshold the socket becomes writable again, applying
// back-pressure to the relay loop instead of piling up data in kernel
// buffers. This reduces per-connection memory and bufferbloat.
func setNotSentLowat(conn syscall.RawConn) {
	conn.Control(func(fd uintptr) { //nolint: errcheck
		unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT, tcpNotSentLowat) //nolint: errcheck
	})
}
