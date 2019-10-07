package hub

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/protocol"
)

type connectionID int

type connection struct {
	conn    conntypes.PacketReadWriteCloser
	mutex   sync.RWMutex
	id      connectionID
	hub     *connectionHub
	pending uint
	closing bool
}

func (c *connection) Write(packet conntypes.Packet) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	err := c.conn.Write(packet)
	if err != nil {
		// if we tried to write into a socket and it was broken, it is
		// a time to reconsider the prescence of this socket at all.
		//
		// probably we need to remove it completely because it seems
		// that connection is broken.
		c.pending = 0
	}
	return err
}

func (c *connection) Read() (conntypes.Packet, error) {
	packet, err := c.conn.Read()

	c.mutex.Lock()
	if err != nil {
		c.pending--
	} else {
		c.pending = 0
	}
	c.mutex.Unlock()

	return packet, err
}

func (c *connection) Stats() (bool, uint) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.closing, c.pending
}

func (c *connection) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.closing = true
	return c.conn.Close()
}

func (c *connection) run() {
	for {
		packet, err := c.conn.Read()
		if err != nil {
			c.Close()
			c.hub.brokenSocketsChan <- c.id
			c.hub = nil
			return
		}

		// TODO
		if channel, ok := Registry.getChannel(conntypes.ConnID{}); ok {
			go channel.write(packet) // nolint: errcheck
		}
	}
}

func newConnection(hub *connectionHub, req *protocol.TelegramRequest) (*connection, error) {
	conn, err := mtproto.TelegramProtocol(req)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new connection: %w", err)
	}

	rv := &connection{
		conn: conn,
		hub:  hub,
		id:   connectionID(rand.Int()),
	}
	go rv.run()

	return rv, nil
}
