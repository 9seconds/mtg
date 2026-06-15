//go:build !linux

package essentials

import "syscall"

func SetRawTCPWindowClamp(_ syscall.RawConn, _ int) error {
	return nil
}
