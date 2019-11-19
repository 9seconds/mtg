package hub

import (
	"context"
	"time"

	"mtg/conntypes"
	"mtg/protocol"
)

const muxGCEvery = time.Minute

type muxNewRequest struct {
	req  *protocol.TelegramRequest
	resp chan<- muxNewResponse
}

type muxNewResponse struct {
	conn *ProxyConn
	err  error
}

type mux struct {
	connections   connectionList
	clients       map[string]*connection
	ctx           context.Context
	channelClosed chan conntypes.ConnID
	channelNew    chan muxNewRequest
}

func (m *mux) run() {
	gcTicker := time.NewTicker(muxGCEvery)
	defer gcTicker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			for _, v := range m.clients {
				v.Close()
			}

			return
		case <-gcTicker.C:
			m.connections.gc()
		case req := <-m.channelNew:
			m.connections.gc()
			proxyConn := newProxyConn(req.req, m.channelClosed)
			conn, err := m.connections.get(proxyConn)

			if err == nil {
				m.clients[string(req.req.ConnID[:])] = conn
			}

			req.resp <- muxNewResponse{
				conn: proxyConn,
				err:  err,
			}
			close(req.resp)
		case connID := <-m.channelClosed:
			if conn, ok := m.clients[string(connID[:])]; ok {
				conn.Detach(connID)
				delete(m.clients, string(connID[:]))
			}
		}
	}
}

func (m *mux) Get(req *protocol.TelegramRequest) (*ProxyConn, error) {
	resp := make(chan muxNewResponse)
	m.channelNew <- muxNewRequest{
		req:  req,
		resp: resp,
	}

	rv := <-resp

	return rv.conn, rv.err
}

func newMux(ctx context.Context) *mux {
	m := &mux{
		ctx:           ctx,
		clients:       make(map[string]*connection),
		channelClosed: make(chan conntypes.ConnID, 1),
		channelNew:    make(chan muxNewRequest),
	}
	go m.run()

	return m
}
