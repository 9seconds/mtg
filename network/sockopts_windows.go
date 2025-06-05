//go:build windows
// +build windows

package network

import "syscall"

func setSocketReuseAddrPort(_ syscall.RawConn) error {
	return nil
}
