package events_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type EventStreamTestSuite struct {
	suite.Suite

	ctx           context.Context
	ctxCancel     context.CancelFunc
	observerMock1 *ObserverMock
	observerMock2 *ObserverMock
	stream        mtglib.EventStream
}

func (suite *EventStreamTestSuite) SetupTest() {
	suite.ctx, suite.ctxCancel = context.WithCancel(context.Background())

	suite.observerMock1 = &ObserverMock{}
	suite.observerMock2 = &ObserverMock{}

	suite.observerMock1.On("Shutdown")
	suite.observerMock2.On("Shutdown")

	factories := make([]events.ObserverFactory, 2)
	factories[0] = func() events.Observer { return suite.observerMock1 }
	factories[1] = func() events.Observer { return suite.observerMock2 }

	suite.stream = events.NewEventStream(factories)
}

func (suite *EventStreamTestSuite) TestEventStart() {
	evt := mtglib.EventStart{
		CreatedAt: time.Now(),
		ConnID:    "connID",
		RemoteIP:  net.ParseIP("10.0.0.1"),
	}

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventStart", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught := args.Get(0).(mtglib.EventStart)

				suite.Equal(evt.CreatedAt, caught.CreatedAt)
				suite.Equal(evt.ConnID, caught.ConnID)
				suite.Equal(evt.RemoteIP.String(), caught.RemoteIP.String())
				suite.Equal(evt.StreamID(), caught.StreamID())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventConnectedToDC() {
	evt := mtglib.EventConnectedToDC{
		CreatedAt: time.Now(),
		ConnID:    "connID",
		RemoteIP:  net.ParseIP("10.0.0.1"),
		DC:        3,
	}

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventConnectedToDC", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught := args.Get(0).(mtglib.EventConnectedToDC)

				suite.Equal(evt.CreatedAt, caught.CreatedAt)
				suite.Equal(evt.ConnID, caught.ConnID)
				suite.Equal(evt.RemoteIP.String(), caught.RemoteIP.String())
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.DC, caught.DC)
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventTraffic() {
	evt := mtglib.EventTraffic{
		CreatedAt: time.Now(),
		ConnID:    "connID",
		Traffic:   1024,
		IsRead:    true,
	}

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventTraffic", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught := args.Get(0).(mtglib.EventTraffic)

				suite.Equal(evt.CreatedAt, caught.CreatedAt)
				suite.Equal(evt.ConnID, caught.ConnID)
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.Traffic, caught.Traffic)
				suite.Equal(evt.IsRead, caught.IsRead)
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventFinish() {
	evt := mtglib.EventFinish{
		CreatedAt: time.Now(),
		ConnID:    "connID",
	}

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventFinish", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught := args.Get(0).(mtglib.EventFinish)

				suite.Equal(evt.CreatedAt, caught.CreatedAt)
				suite.Equal(evt.ConnID, caught.ConnID)
				suite.Equal(evt.StreamID(), caught.StreamID())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventConcurrencyLimited() {
	evt := mtglib.EventConcurrencyLimited{
		CreatedAt: time.Now(),
	}

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventConcurrencyLimited", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught := args.Get(0).(mtglib.EventConcurrencyLimited)

				suite.Equal(evt.CreatedAt, caught.CreatedAt)
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventIPBlocklisted() {
	evt := mtglib.EventIPBlocklisted{
		CreatedAt: time.Now(),
		RemoteIP:  net.ParseIP("10.0.0.10"),
	}

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventIPBlocklisted", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught := args.Get(0).(mtglib.EventIPBlocklisted)

				suite.Equal(evt.CreatedAt, caught.CreatedAt)
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.RemoteIP.String(), caught.RemoteIP.String())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TearDownTest() {
	suite.stream.Shutdown()
	suite.ctxCancel()

	time.Sleep(100 * time.Millisecond)

	suite.observerMock1.AssertExpectations(suite.T())
	suite.observerMock2.AssertExpectations(suite.T())
}

func TestEventStream(t *testing.T) {
	t.Parallel()
	suite.Run(t, &EventStreamTestSuite{})
}
