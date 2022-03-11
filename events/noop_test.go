package events_test

import (
	"context"
	"net"
	"testing"

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
		"start":               mtglib.NewEventStart("connID", net.ParseIP("127.0.0.1")),
		"connected-to-dc":     mtglib.NewEventConnectedToDC("connID", net.ParseIP("127.1.0.1"), 2),
		"domain-fronting":     mtglib.NewEventDomainFronting("connID"),
		"traffic":             mtglib.NewEventTraffic("connID", 1000, true),
		"finish":              mtglib.NewEventFinish("connID"),
		"concurrency-limited": mtglib.NewEventConcurrencyLimited(),
		"ip-blacklisted":      mtglib.NewEventIPBlocklisted(net.ParseIP("10.0.0.10")),
		"replay-attack":       mtglib.NewEventReplayAttack("connID"),
		"ip-list-size":        mtglib.NewEventIPListSize(10, true),
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
}

func (suite *NoopTestSuite) TestObserver() {
	observer := events.NewNoopObserver()

	for name, v := range suite.testData {
		value := v

		suite.T().Run(name, func(t *testing.T) {
			switch typedEvt := value.(type) {
			case mtglib.EventStart:
				observer.EventStart(typedEvt)
			case mtglib.EventConnectedToDC:
				observer.EventConnectedToDC(typedEvt)
			case mtglib.EventDomainFronting:
				observer.EventDomainFronting(typedEvt)
			case mtglib.EventFinish:
				observer.EventFinish(typedEvt)
			case mtglib.EventConcurrencyLimited:
				observer.EventConcurrencyLimited(typedEvt)
			case mtglib.EventIPBlocklisted:
				observer.EventIPBlocklisted(typedEvt)
			case mtglib.EventReplayAttack:
				observer.EventReplayAttack(typedEvt)
			case mtglib.EventIPListSize:
				observer.EventIPListSize(typedEvt)
			}
		})
	}

	observer.Shutdown()
}

func TestNoop(t *testing.T) {
	t.Parallel()
	suite.Run(t, &NoopTestSuite{})
}
