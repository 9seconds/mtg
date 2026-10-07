//go:build !linux

package mtglib

import (
	"net"
	"time"
)

// writeFragmented writes data in one go outside Linux: client-mss needs
// TCP_INFO and tcpi_notsent_bytes, so it works only on Linux.
func writeFragmented(conn net.Conn, data []byte, _ int, _ time.Time) (int, error) {
	return conn.Write(data) //nolint: wrapcheck
}
