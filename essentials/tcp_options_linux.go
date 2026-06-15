//go:build linux

package essentials

import (
	"fmt"
	"syscall"

	"golang.org/x/sys/unix"
)

func SetRawTCPWindowClamp(conn syscall.RawConn, value int) error {
	var err error

	if controlErr := conn.Control(func(fd uintptr) {
		err = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_WINDOW_CLAMP, value)
	}); controlErr != nil && err == nil {
		return fmt.Errorf("cannot access TCP socket: %w", controlErr)
	}
	if err != nil {
		return fmt.Errorf("cannot set TCP_WINDOW_CLAMP: %w", err)
	}

	return nil
}
