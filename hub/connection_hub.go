package hub

import (
	"time"

	"github.com/9seconds/mtg/protocol"
)

const hubGCEvery = time.Minute

type connectionHubRequest struct {
	request  *protocol.TelegramRequest
	response chan<- *connection
}

type connectionHub struct {
	sockets map[int]*connection

	channelBrokenSockets      chan int
	channelConnectionRequests chan *connectionHubRequest
	channelReturnConnections  chan *connection
}

func (c *connectionHub) run() {
	ticker := time.NewTicker(hubGCEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.runGC()
		case request := <-c.channelConnectionRequests:
			c.runConnectionRequest(request)
		case id := <-c.channelBrokenSockets:
			c.runBrokenSocket(id)
		case conn := <-c.channelReturnConnections:
			c.runReturnConnection(conn)
		}
	}
}

func (c *connectionHub) runGC() {
	for key, conn := range c.sockets {
		switch {
		case conn.closed():
			delete(c.sockets, key)
		case conn.idle():
			conn.shutdown()
			delete(c.sockets, key)
			return
		}
	}
}

func (c *connectionHub) runConnectionRequest(req *connectionHubRequest) {
	for key, conn := range c.sockets {
		delete(c.sockets, key)
		if !conn.closed() {
			req.response <- conn
			close(req.response)
			return
		}
	}

	if conn, err := newConnection(req.request, c); err == nil {
		req.response <- conn
	}
	close(req.response)
}

func (c *connectionHub) runBrokenSocket(id int) {
	delete(c.sockets, id)
}

func (c *connectionHub) runReturnConnection(conn *connection) {
	c.sockets[conn.id] = conn
}

func newConnectionHub() *connectionHub {
	rv := &connectionHub{
		sockets:                   map[int]*connection{},
		channelBrokenSockets:      make(chan int, 1),
		channelConnectionRequests: make(chan *connectionHubRequest),
		channelReturnConnections:  make(chan *connection, 1),
	}
	go rv.run()

	return rv
}
