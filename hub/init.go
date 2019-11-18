package hub

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrTimeout = errors.New("timeout")
	ErrClosed  = errors.New("context is closed")

	Hub      Interface
	initOnce sync.Once
)

func Init(ctx context.Context) {
	initOnce.Do(func() {
		Hub = &hub{
			muxes: make(map[int32]*mux),
			ctx:   ctx,
		}
	})
}
