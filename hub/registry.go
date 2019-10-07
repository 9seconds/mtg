package hub

import (
	"context"
	"sync"

	"github.com/9seconds/mtg/conntypes"
)

var Registry *RegistryStruct

type RegistryStruct struct {
	conns map[string]*closeableChannel
	ctx   context.Context
	mutex sync.RWMutex
}

func (r *RegistryStruct) Register(id conntypes.ConnID) ChannelReadCloser {
	channel := newCloseableChannel(r.ctx)

	r.mutex.Lock()
	r.conns[string(id[:])] = channel
	r.mutex.Unlock()

	return channel
}

func (r *RegistryStruct) Unregister(id conntypes.ConnID) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if channel, ok := r.conns[string(id[:])]; ok {
		channel.Close()
		delete(r.conns, string(id[:]))
	}
}

func (r *RegistryStruct) getChannel(id conntypes.ConnID) (*closeableChannel, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if value, ok := r.conns[string(id[:])]; ok {
		return value, true
	}
	return nil, false
}

func InitRegistry(ctx context.Context) {
	Registry = &RegistryStruct{
		ctx:   ctx,
		conns: map[string]*closeableChannel{},
	}
}
