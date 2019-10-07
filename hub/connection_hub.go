package hub

import "time"

const hubGCEvery = time.Minute

type connectionHub struct {
	sockets map[connectionID]*connection

	brokenSocketsChan      chan connectionID
	connectionRequestsChan chan *connectionHubRequest
	returnConnectionsChan  chan *connection
}

func (h *connectionHub) run() {
	gcTicker := time.NewTicker(hubGCEvery)
	defer gcTicker.Stop()

	for {
		select {
		case <-gcTicker.C:
			h.runGC()
		case id := <-h.brokenSocketsChan:
			h.runBrokenConnection(id)
		case request := <-h.connectionRequestsChan:
			h.runConnectionRequest(request)
		case conn := <-h.returnConnectionsChan:
			h.runReturnConnection(conn)
		}
	}
}

func (h *connectionHub) runBrokenConnection(id connectionID) {
	delete(h.sockets, id)
}

func (h *connectionHub) runGC() {
	for key, conn := range h.sockets {
		closing, pending := conn.Stats()
		switch {
		case closing:
			delete(h.sockets, key)
		case pending == 0:
			conn.Close()
			delete(h.sockets, key)
			return
		}

	}
}

func (h *connectionHub) runConnectionRequest(req *connectionHubRequest) {
	for key, conn := range h.sockets {
		closing, _ := conn.Stats()
		delete(h.sockets, key)

		if !closing {
			req.responseChan <- conn
			return
		}
	}

	newConn, err := newConnection(h, req.req)
	if err != nil {
		close(req.responseChan)
		return
	}

	req.responseChan <- newConn
}

func (h *connectionHub) runReturnConnection(conn *connection) {
	h.sockets[conn.id] = conn
}

func newConnectionHub() *connectionHub {
	return &connectionHub{
		sockets: map[connectionID]*connection{},

		brokenSocketsChan:      make(chan connectionID, 1),
		connectionRequestsChan: make(chan *connectionHubRequest),
		returnConnectionsChan:  make(chan *connection, 1),
	}
}
