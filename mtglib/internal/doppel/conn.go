package doppel

import (
	"bytes"
	"context"
	"sync"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

type Conn struct {
	essentials.Conn

	p *connPayload
}

type connPayload struct {
	ctx         context.Context
	ctxCancel   context.CancelCauseFunc
	clock       Clock
	wg          sync.WaitGroup
	writeLock   sync.Mutex
	writeStream bytes.Buffer
}

func (c Conn) Write(p []byte) (int, error) {
	c.p.writeLock.Lock()
	c.p.writeStream.Write(p)
	c.p.writeLock.Unlock()

	return len(p), context.Cause(c.p.ctx)
}

func (c Conn) Start() {
	c.p.wg.Go(func() {
		c.start()
	})
}

func (c Conn) start() {
	buf := [tls.MaxRecordSize]byte{}

	for {
		select {
		case <-c.p.ctx.Done():
			return
		case <-c.p.clock.tick:
		}

		c.p.writeLock.Lock()
		n, err := c.p.writeStream.Read(buf[:c.p.clock.stats.Size()])
		c.p.writeLock.Unlock()

		if n == 0 || err != nil {
			continue
		}

		if err := tls.WriteRecord(c.Conn, buf[:n]); err != nil {
			c.p.ctxCancel(err)
			return
		}
	}
}

func (c Conn) Stop() {
	c.p.ctxCancel(nil)
	c.p.wg.Wait()
}

func NewConn(ctx context.Context, conn essentials.Conn, stats *Stats) Conn {
	ctx, cancel := context.WithCancelCause(ctx)
	rv := Conn{
		Conn: conn,
		p: &connPayload{
			ctx:       ctx,
			ctxCancel: cancel,
			clock: Clock{
				stats: stats,
				tick:  make(chan struct{}),
			},
		},
	}

	rv.p.writeStream.Grow(tls.DefaultBufferSize)

	rv.p.wg.Go(func() {
		rv.p.clock.Start(ctx)
	})
	rv.p.wg.Go(func() {
		rv.start()
	})

	return rv
}
