//go:build !linux

package mtglib

import "syscall"

// acceptedConnErrnos is Linux-specific, see accept_errno_linux.go.
var acceptedConnErrnos []syscall.Errno
