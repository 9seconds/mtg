package rwc

import (
	"context"
	"io"
)

type wrapperPing struct {
	parent      io.ReadWriteCloser
	ctx         context.Context
	channelPing chan<- struct{}
}

func (w *wrapperPing) Read(p []byte) (int, error) {
	n, err := w.parent.Read(p)
	if err == nil {
		select {
		case <-w.ctx.Done():
		case w.channelPing <- struct{}{}:
		}
	}

	return n, err // nolint: wrapcheck
}

func (w *wrapperPing) Write(p []byte) (int, error) {
	n, err := w.parent.Write(p)
	if err == nil {
		select {
		case <-w.ctx.Done():
		case w.channelPing <- struct{}{}:
		}
	}

	return n, err // nolint: wrapcheck
}

func (w *wrapperPing) Close() error {
	return w.parent.Close()
}

func NewPing(ctx context.Context, parent io.ReadWriteCloser, channelPing chan<- struct{}) io.ReadWriteCloser {
	return &wrapperPing{
		parent:      parent,
		ctx:         ctx,
		channelPing: channelPing,
	}
}
