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

type EventConnectedToDC struct {
	CreatedAt time.Time
	ConnID    string
	RemoteIP  net.IP
	DC        int
}

func (e EventConnectedToDC) StreamID() string {
	return e.ConnID
}

type EventTraffic struct {
	CreatedAt time.Time
	ConnID    string
	Traffic   uint
	IsRead    bool
}

func (e EventTraffic) StreamID() string {
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

type EventIPBlocklisted struct {
	CreatedAt time.Time
	RemoteIP  net.IP
}

func (e EventIPBlocklisted) StreamID() string {
	return ""
}
