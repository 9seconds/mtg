package doppel

import (
	"bytes"
	"context"
	"sync"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

var doppelBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, tls.MaxRecordSize)
		return &b
	},
}

type Conn struct {
	essentials.Conn

	p *connPayload
}

type connPayload struct {
	ctx         context.Context
	ctxCancel   context.CancelCauseFunc
	clock       Clock
	wg          sync.WaitGroup
	writeStream bytes.Buffer
	writtenCond sync.Cond
	done        bool
}

func (c Conn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, context.Cause(c.p.ctx)
	}

	c.p.writtenCond.L.Lock()
	c.p.writeStream.Write(p)
	c.p.writtenCond.L.Unlock()

	c.p.writtenCond.Signal()

	return len(p), context.Cause(c.p.ctx)
}

func (c Conn) Start() {
	c.p.wg.Go(func() {
		c.start()
	})
}

func (c Conn) start() {
	bp := doppelBufPool.Get().(*[]byte)
	buf := *bp
	defer doppelBufPool.Put(bp)

	for {
		select {
		case <-c.p.ctx.Done():
			return
		case <-c.p.clock.tick:
		}

		size := c.p.clock.stats.Size()

		c.p.writtenCond.L.Lock()
		for c.p.writeStream.Len() == 0 && !c.p.done {
			c.p.writtenCond.Wait()
		}
		n, _ := c.p.writeStream.Read(buf[tls.SizeHeader : tls.SizeHeader+size])
		c.p.writtenCond.L.Unlock()

		if n == 0 {
			continue
		}

		if err := tls.WriteRecordInPlace(c.Conn, buf, n); err != nil {
			c.p.ctxCancel(err)
			return
		}
	}
}

func (c Conn) Stop() {
	c.p.ctxCancel(nil)

	c.p.writtenCond.L.Lock()
	c.p.done = true
	c.p.writtenCond.L.Unlock()
	c.p.writtenCond.Broadcast()

	c.p.wg.Wait()
}

func NewConn(ctx context.Context, conn essentials.Conn, stats *Stats) Conn {
	ctx, cancel := context.WithCancelCause(ctx)
	rv := Conn{
		Conn: conn,
		p: &connPayload{
			ctx:       ctx,
			ctxCancel: cancel,
			writtenCond: sync.Cond{
				L: &sync.Mutex{},
			},
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
