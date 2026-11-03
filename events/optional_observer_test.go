package events_test

import (
	"context"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const optionalObserverWaitTimeout = 5 * time.Second

// An observer without EventPendingHandshakeLimit still satisfies the required
// interface: adding the optional callback must not break external observers.
var _ events.Observer = &LegacyObserverMock{}

func expectFinish(m *mock.Mock, done chan<- struct{}) {
	m.On("EventFinish", mock.Anything).Once().Run(func(_ mock.Arguments) {
		close(done)
	})
}

func waitClosed(t *testing.T, done <-chan struct{}) {
	t.Helper()

	select {
	case <-done:
	case <-time.After(optionalObserverWaitTimeout):
		t.Fatal("event has not been delivered in time")
	}
}

func TestPendingHandshakeLimitIsOptional(t *testing.T) {
	t.Parallel()

	var legacy events.Observer = &LegacyObserverMock{}

	_, ok := legacy.(events.PendingHandshakeLimitObserver)
	assert.False(t, ok)

	_, ok = events.NewNoopObserver().(events.PendingHandshakeLimitObserver)
	assert.True(t, ok)
}

func TestPendingHandshakeLimitSkipsLegacyObserver(t *testing.T) {
	t.Parallel()

	legacy := &LegacyObserverMock{}
	legacy.On("Shutdown").Maybe()

	done := make(chan struct{})
	expectFinish(&legacy.Mock, done)

	stream := events.NewEventStream([]events.ObserverFactory{
		func() events.Observer { return legacy },
	})
	defer stream.Shutdown()

	// Both events share the stream id, so they go through the same processor
	// in order: EventFinish arriving proves the processor survived the
	// pending-handshake event it could not deliver.
	ctx := context.Background()
	stream.Send(ctx, mtglib.NewEventPendingHandshakeLimit("CONNID", mtglib.PendingHandshakeRejected))
	stream.Send(ctx, mtglib.NewEventFinish("CONNID"))

	waitClosed(t, done)
	legacy.AssertExpectations(t)
}

func TestPendingHandshakeLimitMixedObservers(t *testing.T) {
	t.Parallel()

	legacy := &LegacyObserverMock{}
	legacy.On("Shutdown").Maybe()

	full := &ObserverMock{}
	full.On("Shutdown").Maybe()

	evt := mtglib.NewEventPendingHandshakeLimit("CONNID", mtglib.PendingHandshakeObserved)
	caught := make(chan mtglib.EventPendingHandshakeLimit, 1)

	full.On("EventPendingHandshakeLimit", mock.Anything).Once().Run(func(args mock.Arguments) {
		value, _ := args.Get(0).(mtglib.EventPendingHandshakeLimit)
		caught <- value
	})

	legacyDone := make(chan struct{})
	fullDone := make(chan struct{})

	expectFinish(&legacy.Mock, legacyDone)
	expectFinish(&full.Mock, fullDone)

	// Two factories make the stream fan out through the multi observer.
	stream := events.NewEventStream([]events.ObserverFactory{
		func() events.Observer { return legacy },
		func() events.Observer { return full },
	})
	defer stream.Shutdown()

	ctx := context.Background()
	stream.Send(ctx, evt)
	stream.Send(ctx, mtglib.NewEventFinish("CONNID"))

	waitClosed(t, legacyDone)
	waitClosed(t, fullDone)

	select {
	case value := <-caught:
		require.Equal(t, evt.StreamID(), value.StreamID())
		require.Equal(t, evt.Action, value.Action)
	default:
		t.Fatal("observer with the optional method has not received the event")
	}

	legacy.AssertExpectations(t)
	full.AssertExpectations(t)
}
