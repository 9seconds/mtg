//go:build !linux

package network

import (
	"net"
	"syscall"
)

func setTCPUserTimeout(conn syscall.RawConn, cfg net.KeepAliveConfig) {}
