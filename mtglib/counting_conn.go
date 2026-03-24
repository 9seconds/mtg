package mtglib

import (
	"github.com/9seconds/mtg/v2/essentials"
)

// countingConn wraps essentials.Conn and counts bytes through ProxyStats.
// All methods except Read and Write are delegated to the embedded Conn.
type countingConn struct {
	essentials.Conn
	stats      *ProxyStats
	secretName string
}

func newCountingConn(conn essentials.Conn, stats *ProxyStats, secretName string) *countingConn {
	return &countingConn{Conn: conn, stats: stats, secretName: secretName}
}

func (c *countingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.stats.AddBytesIn(c.secretName, int64(n))
	}

	return n, err
}

func (c *countingConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.stats.AddBytesOut(c.secretName, int64(n))
	}

	return n, err
}
