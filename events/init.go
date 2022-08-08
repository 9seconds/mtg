// Events has a default implementations of EventStream for mtglib.
//
// Please see documentation for [mtglib.EventStream] interface to get an idea
// of such an abstraction. This package has implementations for the default
// event stream.
//
// Default event stream has a list of its own concepts. First, all it does is a
// routing of messages to known observers. It takes an event, defines its type
// and pass this message to a method of the observer.
//
// There might be many observers, but default event stream has a guarantee
// though. It uses StreamID as a sharding key and guarantees that a message
// with the same StreamID will be devlivered to the same observer instance. So,
// each producer is guarateed to get all relevant messages related to the same
// session. It is not possible that it will get EventFinish if it has not seen
// EventStart for that session yet.
package events

import "github.com/9seconds/mtg/v2/mtglib"

// Observer is an instance that listens for the incoming events.
//
// As it is said in the package description, the default event stream
// guarantees that all events with the same StreamID are going to be routed to
// the same instance of the observer. So, there is no need to synchronize
// information about streams between many observers instances, they can have
// their local storage.
type Observer interface {
	// EventStart reacts on incoming mtglib.EventStart event.
	EventStart(mtglib.EventStart)

	// EventFinish reacts on incoming mtglib.EventFinish event.
	EventFinish(mtglib.EventFinish)

	// EventConnectedToDC reacts on incoming mtglib.EventConnectedToDC
	// event.
	EventConnectedToDC(mtglib.EventConnectedToDC)

	// EventDomainFronting reacts on incoming mtglib.EventDomainFronting
	// event.
	EventDomainFronting(mtglib.EventDomainFronting)

	// EventTraffic reacts on incoming mtglib.EventTraffic event.
	EventTraffic(mtglib.EventTraffic)

	// EventConcurrencyLimited reacts on incoming
	// mtglib.EventConcurrencyLimited event.
	EventConcurrencyLimited(mtglib.EventConcurrencyLimited)

	// EventIPBlocklisted reacts on incoming mtglib.EventIPBlocklisted event.
	EventIPBlocklisted(mtglib.EventIPBlocklisted)

	// EventReplayAttack reacts on incoming mtglib.EventReplayAttack event.
	EventReplayAttack(mtglib.EventReplayAttack)

	// EventIPListSize reacts on incoming mtglib.EventIPListSize
	EventIPListSize(mtglib.EventIPListSize)

	// Shutdown stop observer. Default event stream guarantees:
	//   1. If shutdown is executed, it is executed only once
	//   2. Observer won't receieve any new message after this
	//      function call.
	Shutdown()
}

// ObserverFactory creates a new instance of the observer.
//
// Default event stream creates a small set of goroutines to manage incoming
// messages. Each message is routed to an appropriate observer based on a
// sharding key, stream id. So, it is possible that an instance of mtg will
// have many observer instances, not a single one.
type ObserverFactory func() Observer
