//go:build !windows
// +build !windows

package network

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func setSocketReuseAddrPort(conn syscall.RawConn) error {
	var err error

	conn.Control(func(fd uintptr) { //nolint: errcheck
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1) //nolint: nosnakecase
		if err != nil {
			err = fmt.Errorf("cannot set SO_REUSEADDR: %w", err)

			return
		}

		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1) //nolint: nosnakecase
		if err != nil {
			err = fmt.Errorf("cannot set SO_REUSEPORT: %w", err)
		}
	})

	return err
}
