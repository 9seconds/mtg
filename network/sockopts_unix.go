//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package network

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func setReuseAddrPort(rawConn syscall.RawConn, bufferSize int) error {
	var err error
	rawConn.Control(func(fd uintptr) { // nolint: errcheck
		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
		if err != nil {
			err = fmt.Errorf("cannot set SO_REUSEADDR: %w", err)

			return
		}

		err = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if err != nil {
			err = fmt.Errorf("cannot set SO_REUSEPORT: %w", err)
		}
	})

	return err
}
