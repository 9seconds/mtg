//go:build !linux

package mtglib

import (
	"net"
	"time"
)

// writeFragmented writes data in one go outside Linux: client-mss needs
// TCP_INFO and MSG_EOR on TCP, so it works only on Linux.
func writeFragmented(conn net.Conn, data []byte, _, _ int, _ time.Time) (int, error) {
	return conn.Write(data) //nolint: wrapcheck
}
