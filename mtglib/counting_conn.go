package mtglib

import (
	"github.com/dolonet/mtg-multi/essentials"
)

// countingConn wraps essentials.Conn and counts bytes through a cached
// *secretStats pointer. The pointer is resolved once at construction time
// so Read/Write never need to acquire a lock.
type countingConn struct {
	essentials.Conn
	st *secretStats
}

func newCountingConn(conn essentials.Conn, stats *ProxyStats, secretName string) *countingConn {
	return &countingConn{Conn: conn, st: stats.getOrCreate(secretName)}
}

func (c *countingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.st.bytesIn.Add(int64(n))
	}

	return n, err
}

func (c *countingConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.st.bytesOut.Add(int64(n))
	}

	return n, err
}
