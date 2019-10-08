package hub

import (
	"fmt"
	"math/rand"
	"sync"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/protocol"
)

type connection struct {
	conn         conntypes.PacketReadWriteCloser
	mutex        sync.RWMutex
	shutdownOnce sync.Once
	hub          *connectionHub
	id           int
	pending      uint
	done         chan struct{}
}

func (c *connection) read() (conntypes.Packet, error) {
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

func (c *connection) write(packet conntypes.Packet) error {
	err := c.conn.Write(packet)
	if err != nil {
		// if we tried to write into a socket and it was broken, it is
		// a time to reconsider the prescence of this socket at all.
		//
		// probably we need to remove it completely because it seems
		// that connection is broken.
		c.mutex.Lock()
		c.pending = 0
		c.mutex.Unlock()
	}
	return err
}

func (c *connection) shutdown() {
	c.shutdownOnce.Do(func() {
		close(c.done)
		c.hub.channelBrokenSockets <- c.id
	})
}

func (c *connection) closed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *connection) idle() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.pending == 0
}

func (c *connection) run() {
	for {
		packet, err := c.read()
		if err != nil {
			c.shutdown()
			return
		}

		if channel, ok := Registry.getChannel(conntypes.ConnID{}); ok {
			go channel.write(packet) // nolint: errcheck
		}
	}
}

func newConnection(req *protocol.TelegramRequest, hub *connectionHub) (*connection, error) {
	conn, err := mtproto.TelegramProtocol(req)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new connection: %w", err)
	}

	rv := &connection{
		conn: conn,
		hub:  hub,
		id:   rand.Int(),
	}
	go rv.run()

	return rv, nil
}
