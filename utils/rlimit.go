// +build !windows

package utils

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func SetLimits() error {
	rLimit := unix.Rlimit{}
	if err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rLimit); err != nil {
		return fmt.Errorf("cannot get rlimit: %w", err)
	}
	rLimit.Cur = rLimit.Max

	if err := unix.Setrlimit(unix.RLIMIT_NOFILE, &rLimit); err != nil {
		return fmt.Errorf("cannot set rlimit: %w", err)
	}

	return nil
}
