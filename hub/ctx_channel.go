package hub

import (
	"context"
	"time"

	"github.com/9seconds/mtg/mtproto/rpc"
)

const closeableChannelReadTimeout = 2 * time.Minute

type ChannelReadCloser interface {
	Read() (*rpc.ProxyResponse, error)
	Close() error
}

type ctxChannel struct {
	channel chan *rpc.ProxyResponse
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *ctxChannel) Read() (*rpc.ProxyResponse, error) {
	timer := time.NewTimer(closeableChannelReadTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, ErrTimeout
	case <-c.ctx.Done():
		return nil, ErrClosed
	case packet := <-c.channel:
		return packet, nil
	}
}

func (c *ctxChannel) sendBack(response *rpc.ProxyResponse) error {
	select {
	case <-c.ctx.Done():
		return ErrClosed
	case c.channel <- response:
		return nil
	}
}

func (c *ctxChannel) Close() error {
	c.cancel()
	c.channel = nil

	return nil
}

func newCtxChannel(ctx context.Context) *ctxChannel {
	ctx, cancel := context.WithCancel(ctx)

	return &ctxChannel{
		channel: make(chan *rpc.ProxyResponse),
		ctx:     ctx,
		cancel:  cancel,
	}
}
