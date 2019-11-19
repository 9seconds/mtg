package hub

import (
	"sync"
	"time"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/mtproto/rpc"
	"github.com/9seconds/mtg/protocol"
)

const (
	proxyConnWriteTimeout = 2 * time.Minute
	proxyConnReadTimeout  = 2 * time.Minute

	proxyConnBackpressureAfter = 10
)

type ProxyConn struct {
	closeOnce       sync.Once
	req             *protocol.TelegramRequest
	channelResponse chan *rpc.ProxyResponse
	channelClosed   chan<- conntypes.ConnID
	channelWrite    chan<- conntypes.Packet
	channelDone     chan struct{}
}

func (p *ProxyConn) Read() (*rpc.ProxyResponse, error) {
	timer := time.NewTimer(proxyConnReadTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, ErrTimeout
	case <-p.channelDone:
		return nil, ErrClosed
	case packet := <-p.channelResponse:
		return packet, nil
	}
}

func (p *ProxyConn) Write(packet conntypes.Packet) error {
	timer := time.NewTimer(proxyConnWriteTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return ErrTimeout
	case <-p.channelDone:
		return ErrClosed
	case p.channelWrite <- packet:
		return nil
	}
}

func (p *ProxyConn) put(response *rpc.ProxyResponse) {
	select {
	case <-p.channelDone:
	case p.channelResponse <- response:
	}
}

func (p *ProxyConn) Close() {
	p.closeOnce.Do(func() {
		close(p.channelDone)
		go func() {
			p.channelClosed <- p.req.ConnID
		}()
	})
}

func newProxyConn(req *protocol.TelegramRequest, channelClosed chan<- conntypes.ConnID) *ProxyConn {
	return &ProxyConn{
		channelResponse: make(chan *rpc.ProxyResponse, proxyConnBackpressureAfter),
		channelDone:     make(chan struct{}),
		channelClosed:   channelClosed,
		req:             req,
	}
}
