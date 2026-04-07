//go:build !linux && !darwin

package network

import "syscall"

func setNotSentLowat(conn syscall.RawConn) {}
