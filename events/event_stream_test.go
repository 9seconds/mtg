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
	stream        events.EventStream
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
	evt := mtglib.NewEventStart("connID", net.ParseIP("10.0.0.1"))

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventStart", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventStart)

				suite.True(ok)
				suite.Equal(evt.RemoteIP.String(), caught.RemoteIP.String())
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.Timestamp(), caught.Timestamp())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventConnectedToDC() {
	evt := mtglib.NewEventConnectedToDC("connID", net.ParseIP("10.0.0.1"), 3)

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventConnectedToDC", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventConnectedToDC)

				suite.True(ok)
				suite.Equal(evt.RemoteIP.String(), caught.RemoteIP.String())
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.DC, caught.DC)
				suite.Equal(evt.Timestamp(), caught.Timestamp())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventDomainFronting() {
	evt := mtglib.NewEventDomainFronting("connID")

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventDomainFronting", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventDomainFronting)

				suite.True(ok)
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.Timestamp(), caught.Timestamp())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventTraffic() {
	evt := mtglib.NewEventTraffic("connID", 1024, true)

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventTraffic", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventTraffic)

				suite.True(ok)
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.Timestamp(), caught.Timestamp())
				suite.Equal(evt.Traffic, caught.Traffic)
				suite.Equal(evt.IsRead, caught.IsRead)
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventFinish() {
	evt := mtglib.NewEventFinish("connID")

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventFinish", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventFinish)

				suite.True(ok)
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.Timestamp(), caught.Timestamp())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventConcurrencyLimited() {
	evt := mtglib.NewEventConcurrencyLimited()

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventConcurrencyLimited", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventConcurrencyLimited)

				suite.True(ok)
				suite.Equal(evt.Timestamp(), caught.Timestamp())
				suite.Empty(evt.StreamID())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventIPBlocklisted() {
	evt := mtglib.NewEventIPBlocklisted(net.ParseIP("10.0.0.10"))

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventIPBlocklisted", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventIPBlocklisted)

				suite.True(ok)
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.Timestamp(), caught.Timestamp())
				suite.Equal(evt.RemoteIP.String(), caught.RemoteIP.String())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventReplayAttack() {
	evt := mtglib.NewEventReplayAttack("CONNID")

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventReplayAttack", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventReplayAttack)

				suite.True(ok)
				suite.Equal(evt.StreamID(), caught.StreamID())
				suite.Equal(evt.Timestamp(), caught.Timestamp())
			})
	}

	suite.stream.Send(suite.ctx, evt)
	time.Sleep(100 * time.Millisecond)
}

func (suite *EventStreamTestSuite) TestEventIPListSize() {
	evt := mtglib.NewEventIPListSize(10, true)

	for _, v := range []*ObserverMock{suite.observerMock1, suite.observerMock2} {
		v.
			On("EventIPListSize", mock.Anything).
			Once().
			Run(func(args mock.Arguments) {
				caught, ok := args.Get(0).(mtglib.EventIPListSize)

				suite.True(ok)
				suite.Equal(evt.Timestamp(), caught.Timestamp())
				suite.Equal(evt.Size, caught.Size)
				suite.Equal(evt.IsBlockList, caught.IsBlockList)
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
