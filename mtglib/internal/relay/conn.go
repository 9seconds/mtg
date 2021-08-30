package relay

import (
	"fmt"
	"net"
	"time"
)

type conn struct {
	net.Conn
}

func (c conn) Read(p []byte) (int, error) {
	if err := c.SetReadDeadline(time.Now().Add(getTimeout())); err != nil {
		return 0, fmt.Errorf("cannot set read deadline: %w", err)
	}

	return c.Conn.Read(p) // nolint: wrapcheck
}
