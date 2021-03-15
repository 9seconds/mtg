package mtglib

import (
	"net"
	"time"
)

type eventBase struct {
	CreatedAt time.Time
	ConnID    string
}

func (e eventBase) ConnectionID() string {
	return e.ConnID
}

type EventStart struct {
	eventBase

	RemoteIP net.IP
}

type EventFinish struct {
	eventBase
}
