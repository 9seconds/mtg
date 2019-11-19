package hub

import (
	"fmt"
	"sort"

	"mtg/config"
)

type connectionList struct {
	connections []*connection
}

func (c *connectionList) get(conn *ProxyConn) (*connection, error) {
	if len(c.connections) > 0 && c.connections[0].Len() < config.C.MultiplexPerConnection {
		if err := c.connections[0].Attach(conn); err == nil {
			return c.connections[0], nil
		}
	}

	newConn, err := newConnection(conn.req)
	if err != nil {
		return nil, fmt.Errorf("cannot allocate a new connection: %w", err)
	}

	if err = newConn.Attach(conn); err != nil {
		newConn.Close()
		return nil, fmt.Errorf("cannot attach to the newly created connection: %w", err)
	}

	c.connections = append(c.connections, newConn)
	lastIndex := len(c.connections) - 1
	c.connections[0], c.connections[lastIndex] = c.connections[lastIndex], c.connections[0]

	return newConn, nil
}

func (c *connectionList) gc() {
	prevLen := len(c.connections)
	if prevLen == 0 {
		return
	}

	for i := len(c.connections) - 1; i >= 0; i-- {
		lastIndex := len(c.connections) - 1

		if c.connections[i].Done() {
			c.connections[i].Close()

			if len(c.connections)-1 == i {
				c.connections = c.connections[:lastIndex]
			} else {
				c.connections[i], c.connections[lastIndex] = c.connections[lastIndex], c.connections[i]
			}
		}
	}

	if prevLen != len(c.connections) {
		c.sort()
	}
}

func (c *connectionList) sort() {
	if len(c.connections) > 1 {
		sort.Slice(c.connections, func(i, j int) bool {
			return c.connections[i].Len() < c.connections[j].Len()
		})
	}
}
