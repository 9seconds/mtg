package hub

import (
	"context"
	"errors"
	"time"

	"github.com/9seconds/mtg/conntypes"
)

const closeableChannelReadTimeout = 2 * time.Minute

type ChannelReadCloser interface {
	Read() (conntypes.Packet, error)
	Close()
}

type closeableChannel struct {
	channel chan conntypes.Packet
	ctx     context.Context
	cancel  context.CancelFunc
}

func (c *closeableChannel) Read() (conntypes.Packet, error) {
	timer := time.NewTimer(closeableChannelReadTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil, errors.New("timeout")
	case <-c.ctx.Done():
		return nil, errors.New("channel was closed")
	case packet := <-c.channel:
		return packet, nil
	}
}

func (c *closeableChannel) write(packet conntypes.Packet) error {
	select {
	case <-c.ctx.Done():
		return errors.New("channel was closed")
	case c.channel <- packet:
		return nil
	}
}

func (c *closeableChannel) Close() {
	c.cancel()
	c.channel = nil
}

func newCloseableChannel(ctx context.Context) *closeableChannel {
	ctx, cancel := context.WithCancel(ctx)
	return &closeableChannel{
		channel: make(chan conntypes.Packet),
		ctx:     ctx,
		cancel:  cancel,
	}
}
