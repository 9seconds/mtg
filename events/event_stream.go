package events

import (
	"context"
	"math/rand"
	"runtime"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/OneOfOne/xxhash"
)

type EventStream struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	chans     []chan mtglib.Event
}

func (e EventStream) Send(ctx context.Context, evt mtglib.Event) {
	var chanNo uint32

	if streamID := evt.StreamID(); streamID != "" {
		chanNo = xxhash.ChecksumString32(streamID)
	} else {
		chanNo = rand.Uint32()
	}

	select {
	case <-ctx.Done():
	case <-e.ctx.Done():
	case e.chans[int(chanNo)%len(e.chans)] <- evt:
	}
}

func (e EventStream) Shutdown() {
	e.ctxCancel()
}

func NewEventStream(observerFactories []ObserverFactory) EventStream {
	if len(observerFactories) == 0 {
		observerFactories = append(observerFactories, NewNoopObserver)
	}

	ctx, cancel := context.WithCancel(context.Background())
	rv := EventStream{
		ctx:       ctx,
		ctxCancel: cancel,
		chans:     make([]chan mtglib.Event, runtime.NumCPU()),
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		rv.chans[i] = make(chan mtglib.Event, 1)

		if len(observerFactories) == 1 {
			go eventStreamProcessor(ctx, rv.chans[i], observerFactories[0]())
		} else {
			go eventStreamProcessor(ctx, rv.chans[i], newMultiObserver(observerFactories))
		}
	}

	return rv
}

func eventStreamProcessor(ctx context.Context, eventChan <-chan mtglib.Event, observer Observer) { // nolint: cyclop
	defer observer.Shutdown()

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventChan:
			switch typedEvt := evt.(type) {
			case mtglib.EventTraffic:
				observer.EventTraffic(typedEvt)
			case mtglib.EventStart:
				observer.EventStart(typedEvt)
			case mtglib.EventFinish:
				observer.EventFinish(typedEvt)
			case mtglib.EventConnectedToDC:
				observer.EventConnectedToDC(typedEvt)
			case mtglib.EventDomainFronting:
				observer.EventDomainFronting(typedEvt)
			case mtglib.EventIPBlocklisted:
				observer.EventIPBlocklisted(typedEvt)
			case mtglib.EventConcurrencyLimited:
				observer.EventConcurrencyLimited(typedEvt)
			case mtglib.EventReplayAttack:
				observer.EventReplayAttack(typedEvt)
			}
		}
	}
}
