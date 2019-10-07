package hub

import (
	"errors"
	"sync"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/protocol"
)

type Concentrator struct {
	hubs sync.Map
}

func (c *Concentrator) Write(packet conntypes.Packet, req *protocol.TelegramRequest) error {
	hub := c.getHub(req)
	connectionChan := make(chan *connection)
	hub.connectionRequestsChan <- &connectionHubRequest{
		req:          req,
		responseChan: connectionChan,
	}

	conn, ok := <-connectionChan
	if !ok {
		return errors.New("cannot establish connection to telegram")
	}
}

func (c *Concentrator) getHub(req *protocol.TelegramRequest) *connectionHub {
	dcMapRaw, ok := c.hubs.Load(req.ClientProtocol.DC())
	if !ok {
		dcMapRaw, _ = c.hubs.LoadOrStore(req.ClientProtocol.DC(), &sync.Map{})
	}
	dcMap := dcMapRaw.(*sync.Map)

	loaded := true
	hubRaw, ok := dcMap.Load(req.ClientProtocol.ConnectionProtocol())
	if !ok {
		hubRaw, loaded = dcMap.LoadOrStore(req.ClientProtocol.ConnectionProtocol(),
			newConnectionHub())
	}
	hub := hubRaw.(*connectionHub)
	if !loaded {
		go hub.run()
	}

	return hub
}
