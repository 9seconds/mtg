package events

import (
	"context"
	"math/rand"
	"runtime"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/OneOfOne/xxhash"
)

type eventStream struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	chans     []chan mtglib.Event
}

func (e eventStream) Send(ctx context.Context, evt mtglib.Event) {
	var chanNo uint32

	streamID := evt.StreamID()

	if streamID == "" {
		chanNo = rand.Uint32()
	} else {
		chanNo = xxhash.ChecksumString32(streamID)
	}

	select {
	case <-ctx.Done():
	case <-e.ctx.Done():
	case e.chans[int(chanNo)%len(e.chans)] <- evt:
	}
}

func (e eventStream) Shutdown() {
	e.ctxCancel()
}

func NewEventStream(observerFactories []ObserverFactory) mtglib.EventStream {
	if len(observerFactories) == 0 {
		observerFactories = append(observerFactories, NewNoopObserver)
	}

	ctx, cancel := context.WithCancel(context.Background())
	rv := eventStream{
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

func eventStreamProcessor(ctx context.Context, eventChan <-chan mtglib.Event, observer Observer) {
	defer observer.Shutdown()

	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-eventChan:
			switch typedEvt := evt.(type) {
			case mtglib.EventStart:
				observer.EventStart(typedEvt)
			case mtglib.EventFinish:
				observer.EventFinish(typedEvt)
			case mtglib.EventConcurrencyLimited:
				observer.EventConcurrencyLimited(typedEvt)
			}
		}
	}
}
