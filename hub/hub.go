package hub

import (
	"context"
	"sync"

	"github.com/9seconds/mtg/protocol"
)

type hub struct {
	muxes map[int32]*mux
	mutex sync.RWMutex
	ctx   context.Context
}

func (h *hub) Register(req *protocol.TelegramRequest) (*ProxyConn, error) {
	return h.getMux(req).Get(req)
}

func (h *hub) getMux(req *protocol.TelegramRequest) *mux {
	var key int32 = 32767 + int32(req.ClientProtocol.DC()) + 100000*int32(req.ClientProtocol.ConnectionProtocol())

	h.mutex.RLock()
	m, ok := h.muxes[key]
	h.mutex.RUnlock()

	if !ok {
		h.mutex.Lock()
		m, ok = h.muxes[key]

		if !ok {
			m = newMux(h.ctx)
			h.muxes[key] = m
		}

		h.mutex.Unlock()
	}

	return m
}
