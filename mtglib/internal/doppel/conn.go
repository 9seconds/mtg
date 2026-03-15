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
	ctx           context.Context
	ctxCancel     context.CancelCauseFunc
	clock         Clock
	wg            sync.WaitGroup
	syncWriteLock sync.RWMutex
	writeStream   bytes.Buffer
	writeCond     *sync.Cond
}

func (c Conn) Write(p []byte) (int, error) {
	c.p.syncWriteLock.RLock()
	defer c.p.syncWriteLock.RUnlock()

	c.p.writeCond.L.Lock()
	c.p.writeStream.Write(p)
	c.p.writeCond.L.Unlock()

	return len(p), context.Cause(c.p.ctx)
}

func (c Conn) SyncWrite(p []byte) (int, error) {
	c.p.syncWriteLock.Lock()
	defer c.p.syncWriteLock.Unlock()

	c.p.writeCond.L.Lock()
	// wait until buffer is exhausted
	for c.p.writeStream.Len() != 0 && context.Cause(c.p.ctx) == nil {
		c.p.writeCond.Wait()
	}
	c.p.writeStream.Write(p)
	c.p.writeCond.L.Unlock()

	if err := context.Cause(c.p.ctx); err != nil {
		return len(p), err
	}

	c.p.writeCond.L.Lock()
	// wait until data will be sent
	for c.p.writeStream.Len() != 0 && context.Cause(c.p.ctx) == nil {
		c.p.writeCond.Wait()
	}
	c.p.writeCond.L.Unlock()

	return len(p), context.Cause(c.p.ctx)
}

func (c Conn) Start() {
	c.p.wg.Go(func() {
		c.start()
	})
}

func (c Conn) start() {
	defer c.p.writeCond.Broadcast()

	buf := [tls.MaxRecordSize]byte{}

	for {
		select {
		case <-c.p.ctx.Done():
			return
		case <-c.p.clock.tick:
		}

		c.p.writeCond.L.Lock()
		n, err := c.p.writeStream.Read(buf[:c.p.clock.stats.Size()])
		c.p.writeCond.L.Unlock()

		if n == 0 || err != nil {
			continue
		}

		if err := tls.WriteRecord(c.Conn, buf[:n]); err != nil {
			c.p.ctxCancel(err)
			return
		}

		c.p.writeCond.Signal()
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
			writeCond: sync.NewCond(&sync.Mutex{}),
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
