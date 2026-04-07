//go:build linux

package network

import (
	"net"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// Go runtime defaults for KeepAliveConfig when fields are zero.
const (
	goDefaultKeepAliveIdle     = 15 * time.Second
	goDefaultKeepAliveInterval = 15 * time.Second
	goDefaultKeepAliveCount    = 9
)

// setTCPUserTimeout sets TCP_USER_TIMEOUT on a socket. If transmitted
// data remains unacknowledged for this long, the kernel closes the
// connection. As recommended by Cloudflare
// (https://blog.cloudflare.com/when-tcp-sockets-refuse-to-die/),
// the value is computed as: keepidle + keepintvl * keepcnt. This
// ensures TCP_USER_TIMEOUT and keepalives agree on when to give up.
// Best-effort: silently ignored if unsupported.
func setTCPUserTimeout(conn syscall.RawConn, cfg net.KeepAliveConfig) {
	idle := cfg.Idle
	if idle == 0 {
		idle = goDefaultKeepAliveIdle
	}

	interval := cfg.Interval
	if interval == 0 {
		interval = goDefaultKeepAliveInterval
	}

	count := cfg.Count
	if count == 0 {
		count = goDefaultKeepAliveCount
	}

	timeout := idle + interval*time.Duration(count)

	conn.Control(func(fd uintptr) { //nolint: errcheck
		unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_USER_TIMEOUT, int(timeout.Milliseconds())) //nolint: errcheck
	})
}

func setCongestionControl(conn syscall.RawConn) {
	conn.Control(func(fd uintptr) { //nolint: errcheck
		// BBR provides better throughput over lossy and high-latency links compared
		// to the default cubic, which is especially beneficial for mobile and
		// home internet clients. This is best-effort: silently ignored if the
		// kernel does not have tcp_bbr available.
		unix.SetsockoptString(int(fd), unix.IPPROTO_TCP, unix.TCP_CONGESTION, "bbr") //nolint: errcheck
	})
}
