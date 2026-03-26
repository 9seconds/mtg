package doppel

import (
	"bytes"
	"context"
	"sync"
	"time"

	rnd "math/rand/v2"

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
	writeStream bytes.Buffer
	writtenCond sync.Cond
	writeMu     sync.Mutex // protects concurrent writes to underlying Conn
	done        bool
	idlePadding bool
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
	buf := [tls.MaxRecordSize]byte{}

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

		c.p.writeMu.Lock()
		err := tls.WriteRecordInPlace(c.Conn, buf[:], n)
		c.p.writeMu.Unlock()

		if err != nil {
			c.p.ctxCancel(err)
			return
		}
	}
}

// changeCipherSpecRecord is a raw TLS ChangeCipherSpec record.
// Per RFC 8446 §5: extra CCS records after handshake MUST be ignored
// by TLS 1.3 implementations. We use this as idle padding to mimic
// HTTP/2 PING-like keepalive traffic.
var changeCipherSpecRecord = []byte{
	0x14,       // ChangeCipherSpec
	0x03, 0x03, // TLS 1.2
	0x00, 0x01, // length = 1
	0x01,       // value
}

func (c Conn) startIdlePadding() {
	for {
		// Jittered interval: 15-30 seconds, similar to HTTP/2 PING timing.
		interval := 15*time.Second + time.Duration(rnd.IntN(15000))*time.Millisecond

		select {
		case <-c.p.ctx.Done():
			return
		case <-time.After(interval):
		}

		// Only send padding if the buffer is empty (no real data in flight).
		c.p.writtenCond.L.Lock()
		idle := c.p.writeStream.Len() == 0
		c.p.writtenCond.L.Unlock()

		if !idle {
			continue
		}

		c.p.writeMu.Lock()
		_, err := c.Conn.Write(changeCipherSpecRecord)
		c.p.writeMu.Unlock()

		if err != nil {
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

func NewConn(ctx context.Context, conn essentials.Conn, stats *Stats, idlePadding bool) Conn {
	ctx, cancel := context.WithCancelCause(ctx)
	rv := Conn{
		Conn: conn,
		p: &connPayload{
			ctx:         ctx,
			ctxCancel:   cancel,
			idlePadding: idlePadding,
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

	if idlePadding {
		rv.p.wg.Go(func() {
			rv.startIdlePadding()
		})
	}

	return rv
}
