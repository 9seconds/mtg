//go:build linux

package network

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// setCongestionControl sets BBR as the TCP congestion control algorithm.
// BBR provides better throughput over lossy and high-latency links compared
// to the default cubic, which is especially beneficial for mobile and
// home internet clients. This is best-effort: silently ignored if the
// kernel does not have tcp_bbr available.
func setCongestionControl(conn syscall.RawConn) {
	conn.Control(func(fd uintptr) { //nolint: errcheck
		unix.SetsockoptString(int(fd), unix.IPPROTO_TCP, unix.TCP_CONGESTION, "bbr") //nolint: errcheck
	})
}
