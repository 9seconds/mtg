package hub

import (
	"context"
	"time"

	"github.com/9seconds/mtg/conntypes"
)

const closeableChannelReadTimeout = 2 * time.Minute

type ChannelReadCloser interface {
	Read() (conntypes.Packet, error)
	Close() error
}

type ctxChannel struct {
	channel chan conntypes.Packet
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *ctxChannel) Read() (conntypes.Packet, error) {
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

func (c *ctxChannel) write(packet conntypes.Packet) error {
	select {
	case <-c.ctx.Done():
		return ErrClosed
	case c.channel <- packet:
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
		channel: make(chan conntypes.Packet),
		ctx:     ctx,
		cancel:  cancel,
	}
}
