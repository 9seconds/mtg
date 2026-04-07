//go:build windows

package network

import "syscall"

func setSocketReuseAddrPort(conn syscall.RawConn) error {
	return nil
}
