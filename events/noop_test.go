package events_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/suite"
)

type NoopTestSuite struct {
	suite.Suite

	testData map[string]mtglib.Event
	ctx      context.Context
}

func (suite *NoopTestSuite) SetupSuite() {
	suite.testData = map[string]mtglib.Event{
		"start": mtglib.EventStart{
			CreatedAt: time.Now(),
			ConnID:    "connID",
			RemoteIP:  net.ParseIP("127.0.0.1"),
		},
		"finish": mtglib.EventFinish{
			CreatedAt: time.Now(),
			ConnID:    "connID",
		},
		"concurrency-limited": mtglib.EventConcurrencyLimited{},
	}
	suite.ctx = context.Background()
}

func (suite *NoopTestSuite) TestStream() {
	stream := events.NewNoopStream()

	for name, v := range suite.testData {
		value := v

		suite.T().Run(name, func(t *testing.T) {
			stream.Send(suite.ctx, value)
		})
	}

	stream.Shutdown()
}

func (suite *NoopTestSuite) TestObserver() {
	observer := events.NewNoopObserver()

	for name, v := range suite.testData {
		value := v

		suite.T().Run(name, func(t *testing.T) {
			switch typedEvt := value.(type) {
			case mtglib.EventStart:
				observer.EventStart(typedEvt)
			case mtglib.EventFinish:
				observer.EventFinish(typedEvt)
			case mtglib.EventConcurrencyLimited:
				observer.EventConcurrencyLimited(typedEvt)
			}
		})
	}

	observer.Shutdown()
}

func TestNoop(t *testing.T) {
	t.Parallel()
	suite.Run(t, &NoopTestSuite{})
}
