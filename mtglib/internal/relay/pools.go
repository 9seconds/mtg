package relay

import (
	"context"
	"sync"
	"time"
)

var relayPool = sync.Pool{
	New: func() interface{} {
		return &Relay{
			tickChannel:  make(chan struct{}),
			errorChannel: make(chan error, 1),
		}
	},
}

func AcquireRelay(ctx context.Context, logger Logger, bufferSize int, idleTimeout time.Duration) *Relay {
	ctx, cancel := context.WithCancel(ctx)

	r, ok := relayPool.Get().(*Relay)
	if !ok {
		panic("Relay pool has no relay!")
	}

	r.ctx = ctx
	r.ctxCancel = cancel
	r.logger = logger
	r.tickTimeout = idleTimeout

	if len(r.eastBuffer) != bufferSize {
		r.eastBuffer = make([]byte, bufferSize)
	}

	if len(r.westBuffer) != bufferSize {
		r.westBuffer = make([]byte, bufferSize)
	}

	return r
}

func ReleaseRelay(r *Relay) {
	r.Reset()
	relayPool.Put(r)
}
