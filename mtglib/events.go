package mtglib

import (
	"net"
	"time"
)

type EventStart struct {
	CreatedAt time.Time
	ConnID    string
	RemoteIP  net.IP
}

func (e EventStart) StreamID() string {
	return e.ConnID
}

type EventFinish struct {
	CreatedAt time.Time
	ConnID    string
}

func (e EventFinish) StreamID() string {
	return e.ConnID
}

type EventConcurrencyLimited struct {
	CreatedAt time.Time
}

func (e EventConcurrencyLimited) StreamID() string {
	return ""
}
