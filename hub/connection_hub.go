package hub

import (
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/protocol"
)

const hubGCEvery = time.Minute

type connectionHubRequest struct {
	request  *protocol.TelegramRequest
	response chan<- *connection
}

type connectionHub struct {
	sockets map[int]*connection
	logger  *zap.SugaredLogger

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
	logger := c.logger.Named("gc")

	for key, conn := range c.sockets {
		switch {
		case conn.closed():
			logger.Debugw("Delete closed socket", "key", key)
			delete(c.sockets, key)
		case conn.idle():
			logger.Debugw("Delete idle socket", "key", key)
			conn.shutdown()
			delete(c.sockets, key)

			return
		}
	}
}

func (c *connectionHub) runConnectionRequest(req *connectionHubRequest) {
	logger := c.logger.Named("request").With("connection-id", req.request.ConnID)

	for key, conn := range c.sockets {
		delete(c.sockets, key)

		if !conn.closed() {
			logger.Debugw("Choose connection",
				"id", conn.id,
				"remote_addr", conn.conn.RemoteAddr())
			req.response <- conn
			close(req.response)

			return
		}
	}

	if conn, err := newConnection(req.request, c); err == nil {
		logger.Debugw("New connection",
			"id", conn.id,
			"remote_addr", conn.conn.RemoteAddr())
		req.response <- conn
	}

	close(req.response)
}

func (c *connectionHub) runBrokenSocket(id int) {
	c.logger.Named("broken-socket").Debugw("Delete broken socket", "id", id)
	delete(c.sockets, id)
}

func (c *connectionHub) runReturnConnection(conn *connection) {
	c.logger.Named("return-connection").Debugw("Return connection",
		"id", conn.id,
		"remote_addr", conn.conn.RemoteAddr())

	c.sockets[conn.id] = conn
}

func newConnectionHub(logger *zap.SugaredLogger) *connectionHub {
	rv := &connectionHub{
		logger:                    logger.Named("connection-hub"),
		sockets:                   map[int]*connection{},
		channelBrokenSockets:      make(chan int, 1),
		channelConnectionRequests: make(chan *connectionHubRequest),
		channelReturnConnections:  make(chan *connection, 1),
	}
	go rv.run()

	return rv
}
