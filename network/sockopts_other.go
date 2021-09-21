//go:build windows || js
// +build windows js

package network

import "syscall"

func setReuseAddrPort(syscall.RawConn, int) error {
	return nil
}
