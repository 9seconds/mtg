package network

import "net"

// applyKeepAlive enables (or disables) TCP keepalive on conn.
//
// OpenBSD has no user-settable per-socket TCP keepalive options: TCP_KEEPIDLE,
// TCP_KEEPINTVL and TCP_KEEPCNT do not exist on OpenBSD, and Go's
// (*TCPConn).SetKeepAliveConfig therefore returns ENOPROTOOPT ("protocol not
// available") for any non-negative Idle/Interval/Count value (see
// src/net/tcpsockopt_openbsd.go in the Go source tree). Calling
// SetKeepAliveConfig with mtg's defaults (zero values) breaks every accepted
// listener connection and every outbound dial on OpenBSD.
//
// On OpenBSD we only flip SO_KEEPALIVE on or off; the keepalive timing is
// controlled system-wide via the sysctl knobs net.inet.tcp.keepidle and
// net.inet.tcp.keepintvl.
func applyKeepAlive(conn *net.TCPConn, cfg net.KeepAliveConfig) error {
	return conn.SetKeepAlive(cfg.Enable) //nolint: wrapcheck
}
