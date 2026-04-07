//go:build !linux

package network

import (
	"net"
	"syscall"
)

func setCongestionControl(conn syscall.RawConn)                       {}
func setTCPUserTimeout(conn syscall.RawConn, cfg net.KeepAliveConfig) {}
