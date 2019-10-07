package hub

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/protocol"
)

type hub struct {
	subs  map[string]*connectionHub
	mutex sync.RWMutex
}

func (h *hub) Write(packet conntypes.Packet, req *protocol.TelegramRequest) error {
	sub := h.getHub(req)
	connections := make(chan *connection)
	sub.channelConnectionRequests <- &connectionHubRequest{
		request:  req,
		response: connections,
	}

	conn, ok := <-connections
	if !ok {
		return ErrCannotCreateConnection
	}

	if err := conn.write(packet); err != nil {
		return fmt.Errorf("cannot send packet: %w", err)
	}
	return nil
}

func (h *hub) getHub(req *protocol.TelegramRequest) *connectionHub {
	keyBuilder := strings.Builder{}
	binary.Write(&keyBuilder, binary.LittleEndian, int16(req.ClientProtocol.DC()))
	keyBuilder.WriteRune('_')
	binary.Write(&keyBuilder, binary.LittleEndian, uint8(req.ClientProtocol.ConnectionProtocol()))
	key := keyBuilder.String()

	h.mutex.RLock()
	rv, ok := h.subs[key]
	h.mutex.RUnlock()

	if !ok {
		h.mutex.Lock()
		defer h.mutex.Unlock()

		rv, ok = h.subs[key]
		if !ok {
			rv = newConnectionHub()
			h.subs[key] = rv
		}
	}

	return rv
}
