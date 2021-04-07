package mtglib

import (
	"net"
	"time"
)

type eventBase struct {
	streamID  string
	timestamp time.Time
}

func (e eventBase) StreamID() string {
	return e.streamID
}

func (e eventBase) Timestamp() time.Time {
	return e.timestamp
}

type EventStart struct {
	eventBase

	RemoteIP net.IP
}

type EventConnectedToDC struct {
	eventBase

	RemoteIP net.IP
	DC       int
}

type EventTraffic struct {
	eventBase

	Traffic uint
	IsRead  bool
}

type EventFinish struct {
	eventBase
}

type EventDomainFronting struct {
	eventBase
}

type EventConcurrencyLimited struct {
	eventBase
}

type EventIPBlocklisted struct {
	eventBase

	RemoteIP net.IP
}

type EventReplayAttack struct {
	eventBase
}

func NewEventStart(streamID string, remoteIP net.IP) EventStart {
	return EventStart{
		eventBase: eventBase{
			timestamp: time.Now(),
			streamID:  streamID,
		},
		RemoteIP: remoteIP,
	}
}

func NewEventConnectedToDC(streamID string, remoteIP net.IP, dc int) EventConnectedToDC {
	return EventConnectedToDC{
		eventBase: eventBase{
			timestamp: time.Now(),
			streamID:  streamID,
		},
		RemoteIP: remoteIP,
		DC:       dc,
	}
}

func NewEventTraffic(streamID string, traffic uint, isRead bool) EventTraffic {
	return EventTraffic{
		eventBase: eventBase{
			timestamp: time.Now(),
			streamID:  streamID,
		},
		Traffic: traffic,
		IsRead:  isRead,
	}
}

func NewEventFinish(streamID string) EventFinish {
	return EventFinish{
		eventBase: eventBase{
			timestamp: time.Now(),
			streamID:  streamID,
		},
	}
}

func NewEventDomainFronting(streamID string) EventDomainFronting {
	return EventDomainFronting{
		eventBase: eventBase{
			timestamp: time.Now(),
			streamID:  streamID,
		},
	}
}

func NewEventConcurrencyLimited() EventConcurrencyLimited {
	return EventConcurrencyLimited{
		eventBase: eventBase{
			timestamp: time.Now(),
		},
	}
}

func NewEventIPBlocklisted(remoteIP net.IP) EventIPBlocklisted {
	return EventIPBlocklisted{
		eventBase: eventBase{
			timestamp: time.Now(),
		},
		RemoteIP: remoteIP,
	}
}

func NewEventReplayAttack(streamID string) EventReplayAttack {
	return EventReplayAttack{
		eventBase: eventBase{
			timestamp: time.Now(),
			streamID:  streamID,
		},
	}
}
