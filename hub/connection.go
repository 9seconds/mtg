package hub

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/protocol"
)

const connectionTTL = time.Hour

type connection struct {
	conn            conntypes.PacketReadWriteCloser
	proxyConns      map[string]*ProxyConn
	closeOnce       sync.Once
	proxyConnsMutex sync.RWMutex
	id              int
	logger          *zap.SugaredLogger

	channelDone       chan struct{}
	channelWrite      chan conntypes.Packet
	channelRead       chan *rpc.ProxyResponse
	channelConnAttach chan *ProxyConn
	channelConnDetach chan conntypes.ConnID
}

func (c *connection) run() {
	defer c.Close()

	ttl := time.NewTimer(connectionTTL)
	defer ttl.Stop()

	for {
		select {
		case <-c.channelDone:
			for _, v := range c.proxyConns {
				v.Close()
			}

			return
		case <-ttl.C:
			c.logger.Debugw("Closing connection by TTL")
			c.Close()
		case resp := <-c.channelRead:
			if channel, ok := c.proxyConns[string(resp.ConnID[:])]; ok {
				if resp.Type == rpc.ProxyResponseTypeCloseExt {
					channel.Close()
				} else {
					channel.put(resp)
				}
			}
		case packet := <-c.channelWrite:
			if err := c.conn.Write(packet); err != nil {
				c.logger.Debugw("Cannot write packet", "error", err)
				c.Close()
			}
		case conn := <-c.channelConnAttach:
			c.proxyConnsMutex.Lock()
			c.proxyConns[string(conn.req.ConnID[:])] = conn
			c.proxyConnsMutex.Unlock()
			conn.channelWrite = c.channelWrite
		case connID := <-c.channelConnDetach:
			if conn, ok := c.proxyConns[string(connID[:])]; ok {
				c.proxyConnsMutex.Lock()
				delete(c.proxyConns, string(connID[:]))
				c.proxyConnsMutex.Unlock()
				conn.Close()
			}
		}
	}
}

func (c *connection) readLoop() {
	for {
		packet, err := c.conn.Read()
		if err != nil {
			c.logger.Debugw("Cannot read packet", "error", err)
			c.Close()

			return
		}

		response, err := rpc.ParseProxyResponse(packet)
		if err != nil {
			c.logger.Debugw("Failed response", "error", err)
			continue
		}

		select {
		case <-c.channelDone:
			return
		case c.channelRead <- response:
		}
	}
}

func (c *connection) Close() {
	c.closeOnce.Do(func() {
		c.logger.Debugw("Closing connection")

		close(c.channelDone)
		c.conn.Close()
	})
}

func (c *connection) Done() bool {
	select {
	case <-c.channelDone:
		return true
	default:
		return c.Len() == 0
	}
}

func (c *connection) Len() int {
	c.proxyConnsMutex.RLock()
	defer c.proxyConnsMutex.RUnlock()

	return len(c.proxyConns)
}

func (c *connection) Attach(conn *ProxyConn) error {
	select {
	case <-c.channelDone:
		return ErrClosed
	case c.channelConnAttach <- conn:
		return nil
	}
}

func (c *connection) Detach(connID conntypes.ConnID) {
	select {
	case <-c.channelDone:
	case c.channelConnDetach <- connID:
	}
}

func newConnection(req *protocol.TelegramRequest) (*connection, error) {
	conn, err := mtproto.TelegramProtocol(req)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new connection: %w", err)
	}

	id := rand.Int() // nolint: gosec
	rv := &connection{
		conn: conn,
		id:   id,
		logger: zap.S().Named("hub-connection").With("id", id,
			"dc", req.ClientProtocol.DC(),
			"protocol", req.ClientProtocol.ConnectionProtocol()),
		proxyConns: make(map[string]*ProxyConn),

		channelRead:       make(chan *rpc.ProxyResponse, 1),
		channelDone:       make(chan struct{}),
		channelWrite:      make(chan conntypes.Packet),
		channelConnAttach: make(chan *ProxyConn),
		channelConnDetach: make(chan conntypes.ConnID),
	}

	go rv.readLoop()

	go rv.run()

	return rv, nil
}
