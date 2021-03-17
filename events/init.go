package events

import "github.com/9seconds/mtg/v2/mtglib"

type Observer interface {
	EventStart(mtglib.EventStart)
	EventFinish(mtglib.EventFinish)
	EventConcurrencyLimited(mtglib.EventConcurrencyLimited)
	EventIPBlocklisted(mtglib.EventIPBlocklisted)

	Shutdown()
}

type ObserverFactory func() Observer
