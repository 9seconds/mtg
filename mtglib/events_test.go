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
}

func (suite *EventsTestSuite) TestEventFinish() {
	evt := mtglib.EventFinish{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
	}

	suite.Equal("CONNID", evt.StreamID())
}

func (suite *EventsTestSuite) TestEventConnectedToDC() {
	evt := mtglib.EventConnectedToDC{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
		RemoteIP:  net.ParseIP("10.0.0.10"),
		DC:        3,
	}

	suite.Equal("CONNID", evt.StreamID())
}

func (suite *EventsTestSuite) TestEventTraffic() {
	evt := mtglib.EventTraffic{
		CreatedAt: time.Now(),
		ConnID:    "CONNID",
		Traffic:   3,
		IsRead:    true,
	}

	suite.Equal("CONNID", evt.StreamID())
}

func (suite *EventsTestSuite) TestEventConcurrencyLimited() {
	suite.Empty(mtglib.EventConcurrencyLimited{}.StreamID())
}

func (suite *EventsTestSuite) TestEventIPBlocklisted() {
	suite.Empty(mtglib.EventIPBlocklisted{}.StreamID())
}

func TestEvents(t *testing.T) {
	t.Parallel()
	suite.Run(t, &EventsTestSuite{})
}
