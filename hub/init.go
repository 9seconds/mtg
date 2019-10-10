package hub

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"
)

var (
	Registry *registry
	Hub      *hub

	ErrTimeout                = errors.New("timeout")
	ErrClosed                 = errors.New("channel was closed")
	ErrCannotCreateConnection = errors.New("cannot create connection")

	initOnce sync.Once
)

func Init(ctx context.Context) {
	initOnce.Do(func() {
		Registry = &registry{
			conns: map[string]*ctxChannel{},
			ctx:   ctx,
		}
		Hub = &hub{
			subs:   map[string]*connectionHub{},
			logger: zap.S().Named("hub"),
		}
	})
}
