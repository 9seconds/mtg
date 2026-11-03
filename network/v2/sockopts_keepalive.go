//go:build !openbsd

package network

import "net"

// applyKeepAlive enables TCP keepalive on conn and applies the per-socket
// idle/interval/count tuning from cfg.
func applyKeepAlive(conn *net.TCPConn, cfg net.KeepAliveConfig) error {
	return conn.SetKeepAliveConfig(cfg) //nolint: wrapcheck
}
