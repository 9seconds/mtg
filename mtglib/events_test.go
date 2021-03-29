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
	evt := mtglib.EventStart{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
		RemoteIP:  net.ParseIP("10.0.0.10"),
	}

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventFinish() {
	evt := mtglib.EventFinish{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
	}

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventConnectedToDC() {
	evt := mtglib.EventConnectedToDC{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
		RemoteIP:  net.ParseIP("10.0.0.10"),
		DC:        3,
	}

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventTraffic() {
	evt := mtglib.EventTraffic{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
		Traffic:   3,
		IsRead:    true,
	}

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventDomainFronting() {
	evt := mtglib.EventDomainFronting{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
	}

	suite.Equal("CONNID", evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventConcurrencyLimited() {
	evt := mtglib.EventConcurrencyLimited{
		CreatedAt: time.Now(),
	}

	suite.Empty(evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func (suite *EventsTestSuite) TestEventIPBlocklisted() {
	evt := mtglib.EventIPBlocklisted{
		CreatedAt: time.Now(),
		RemoteIP:  net.ParseIP("10.0.0.10"),
	}

	suite.Empty(evt.StreamID())
	suite.WithinDuration(time.Now(), evt.Timestamp(), 10*time.Millisecond)
}

func TestEvents(t *testing.T) {
	t.Parallel()
	suite.Run(t, &EventsTestSuite{})
}
