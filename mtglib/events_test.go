package mtglib_test

import (
	"net"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/suite"
)

type EventsTestSuite struct {
	suite.Suite
}

func (suite *EventsTestSuite) TestEventStart() {
	evt := mtglib.NewEventStart("CONNID", net.ParseIP("10.0.0.10"))

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventFinish() {
	evt := mtglib.NewEventFinish("CONNID")

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventConnectedToDC() {
	evt := mtglib.NewEventConnectedToDC("CONNID", net.ParseIP("10.0.0.10"), 3)

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventTraffic() {
	evt := mtglib.NewEventTraffic("CONNID", 1000, true)

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventDomainFronting() {
	evt := mtglib.NewEventDomainFronting("CONNID")

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventConcurrencyLimited() {
	evt := mtglib.NewEventConcurrencyLimited()

	suite.Empty(evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventIPBlocklisted() {
	evt := mtglib.NewEventIPBlocklisted(net.ParseIP("10.0.0.10"))

	suite.Empty(evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
	suite.True(evt.IsBlockList)
}

func (suite *EventsTestSuite) TestEventIPAllowlisted() {
	evt := mtglib.NewEventIPAllowlisted(net.ParseIP("10.0.0.10"))

	suite.Empty(evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
	suite.False(evt.IsBlockList)
}

func (suite *EventsTestSuite) TestEventReplayAttack() {
	evt := mtglib.NewEventReplayAttack("CONNID")

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventIPListSize() {
	evt := mtglib.NewEventIPListSize(10, false)

	suite.Empty(evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
	suite.Equal(10, evt.Size)
	suite.False(evt.IsBlockList)
}

func TestEvents(t *testing.T) {
	t.Parallel()
	suite.Run(t, &EventsTestSuite{})
}
