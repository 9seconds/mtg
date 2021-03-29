package events

import "github.com/9seconds/mtg/v2/mtglib"

type Observer interface {
	EventStart(mtglib.EventStart)
	EventFinish(mtglib.EventFinish)
	EventConnectedToDC(mtglib.EventConnectedToDC)
	EventDomainFronting(mtglib.EventDomainFronting)
	EventTraffic(mtglib.EventTraffic)
	EventConcurrencyLimited(mtglib.EventConcurrencyLimited)
	EventIPBlocklisted(mtglib.EventIPBlocklisted)
	EventReplayAttack(mtglib.EventReplayAttack)

	Shutdown()
}

type ObserverFactory func() Observer
