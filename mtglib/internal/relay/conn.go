package relay

import "io"

type conn struct {
	io.ReadWriteCloser

	relay *Relay
}

func (c conn) Read(p []byte) (int, error) {
	n, err := c.ReadWriteCloser.Read(p)

	select {
	case <-c.relay.ctx.Done():
	case c.relay.tickChannel <- struct{}{}:
	}

	return n, err // nolint: wrapcheck
}

func (c conn) Write(p []byte) (int, error) {
	n, err := c.ReadWriteCloser.Write(p)

	select {
	case <-c.relay.ctx.Done():
	case c.relay.tickChannel <- struct{}{}:
	}

	return n, err // nolint: wrapcheck
}
