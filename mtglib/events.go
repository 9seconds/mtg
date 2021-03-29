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

func (e EventStart) Timestamp() time.Time {
	return e.CreatedAt
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

func (e EventConnectedToDC) Timestamp() time.Time {
	return e.CreatedAt
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

func (e EventTraffic) Timestamp() time.Time {
	return e.CreatedAt
}

type EventFinish struct {
	CreatedAt time.Time
	ConnID    string
}

func (e EventFinish) StreamID() string {
	return e.ConnID
}

func (e EventFinish) Timestamp() time.Time {
	return e.CreatedAt
}

type EventDomainFronting struct {
	CreatedAt time.Time
	ConnID    string
}

func (e EventDomainFronting) StreamID() string {
	return e.ConnID
}

func (e EventDomainFronting) Timestamp() time.Time {
	return e.CreatedAt
}

type EventConcurrencyLimited struct {
	CreatedAt time.Time
}

func (e EventConcurrencyLimited) StreamID() string {
	return ""
}

func (e EventConcurrencyLimited) Timestamp() time.Time {
	return e.CreatedAt
}

type EventIPBlocklisted struct {
	CreatedAt time.Time
	RemoteIP  net.IP
}

func (e EventIPBlocklisted) StreamID() string {
	return ""
}

func (e EventIPBlocklisted) Timestamp() time.Time {
	return e.CreatedAt
}

type EventReplayAttack struct {
	CreatedAt time.Time
	ConnID    string
}

func (e EventReplayAttack) StreamID() string {
	return e.ConnID
}

func (e EventReplayAttack) Timestamp() time.Time {
	return e.CreatedAt
}
