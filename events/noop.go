package events

import (
	"context"

	"github.com/9seconds/mtg/v2/mtglib"
)

type noop struct{}

func (n noop) Send(ctx context.Context, evt mtglib.Event) {}

// NewNoopStream creates a stream which discards each message.
func NewNoopStream() mtglib.EventStream {
	return noop{}
}

type noopObserver struct{}

func (n noopObserver) EventStart(_ mtglib.EventStart)                           {}
func (n noopObserver) EventConnectedToDC(_ mtglib.EventConnectedToDC)           {}
func (n noopObserver) EventDomainFronting(_ mtglib.EventDomainFronting)         {}
func (n noopObserver) EventTraffic(_ mtglib.EventTraffic)                       {}
func (n noopObserver) EventFinish(_ mtglib.EventFinish)                         {}
func (n noopObserver) EventConcurrencyLimited(_ mtglib.EventConcurrencyLimited) {}
func (n noopObserver) EventIPBlocklisted(_ mtglib.EventIPBlocklisted)           {}
func (n noopObserver) EventReplayAttack(_ mtglib.EventReplayAttack)             {}
func (n noopObserver) EventIPListSize(_ mtglib.EventIPListSize)                 {}
func (n noopObserver) Shutdown()                                                {}

// NewNoopObserver creates an observer which discards each message.
func NewNoopObserver() Observer {
	return noopObserver{}
}
