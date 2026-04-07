//go:build !linux

package network

import "syscall"

func setCongestionControl(conn syscall.RawConn) {}
