package hub

import (
	"context"
	"sync"

	"github.com/9seconds/mtg/conntypes"
)

type registry struct {
	conns map[string]*ctxChannel
	ctx   context.Context
	mutex sync.RWMutex
}

func (r *registry) Register(id conntypes.ConnID) ChannelReadCloser {
	channel := newCtxChannel(r.ctx)

	r.mutex.Lock()
	r.conns[string(id[:])] = channel
	r.mutex.Unlock()

	return channel
}

func (r *registry) Unregister(id conntypes.ConnID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if channel, ok := r.conns[string(id[:])]; ok {
		channel.Close()
		delete(r.conns, string(id[:]))
	}
}

func (r *registry) getChannel(id conntypes.ConnID) (*ctxChannel, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if value, ok := r.conns[string(id[:])]; ok {
		return value, true
	}
	return nil, false
}

func InitRegistry(ctx context.Context) {
	Registry = &registry{
		ctx:   ctx,
		conns: map[string]*ctxChannel{},
	}
}
