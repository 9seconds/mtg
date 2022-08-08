package stats_test

import (
	"bytes"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/stats"
	statsd "github.com/smira/go-statsd"
	"github.com/stretchr/testify/suite"
)

const statsdSleepTime = 4 * statsd.DefaultFlushInterval

type statsdFakeServer struct {
	conn  *net.UDPConn
	buf   *bytes.Buffer
	mutex sync.Mutex
}

func (s *statsdFakeServer) Addr() string {
	return s.conn.LocalAddr().String()
}

func (s *statsdFakeServer) Close() error {
	if s.conn != nil {
		return s.conn.Close() //nolint: wrapcheck
	}

	return nil
}

func (s *statsdFakeServer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return strings.TrimSpace(s.buf.String())
}

func statsdNewFakeServer() *statsdFakeServer {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	if err != nil {
		panic(err)
	}

	rv := &statsdFakeServer{
		conn: conn,
		buf:  &bytes.Buffer{},
	}

	go func() {
		currentBuffer := make([]byte, 4096)

		for {
			n, _, err := conn.ReadFromUDP(currentBuffer)
			if n > 0 {
				rv.mutex.Lock()
				rv.buf.Write(currentBuffer[:n])
				rv.mutex.Unlock()
			}

			if err != nil {
				return
			}
		}
	}()

	return rv
}

type StatsdTestSuite struct {
	suite.Suite

	statsdServer *statsdFakeServer
	factory      stats.StatsdFactory
	statsd       events.Observer
}

func (suite *StatsdTestSuite) SetupTest() {
	suite.statsdServer = statsdNewFakeServer()

	factory, err := stats.NewStatsd(suite.statsdServer.Addr(),
		logger.NewNoopLogger(), "mtg.", "datadog")
	if err != nil {
		panic(err)
	}

	suite.factory = factory
	suite.statsd = suite.factory.Make()
}

func (suite *StatsdTestSuite) TearDownTest() {
	suite.statsd.Shutdown()
	suite.factory.Close()
	suite.statsdServer.Close()
}

func (suite *StatsdTestSuite) TestTelegramPath() {
	suite.statsd.EventStart(
		mtglib.NewEventStart("connID", net.ParseIP("10.0.0.10")))
	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.client_connections:+1|g|#ip_family:ipv4", suite.statsdServer.String())

	suite.statsd.EventConnectedToDC(
		mtglib.NewEventConnectedToDC("connID", net.ParseIP("10.1.0.10"), 2))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(),
		"mtg.telegram_connections:+1|g|#telegram_ip:10.1.0.10,dc:2")

	suite.statsd.EventTraffic(
		mtglib.NewEventTraffic("connID", 30, true))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(),
		"mtg.telegram_traffic:30|c|#telegram_ip:10.1.0.10,dc:2,direction:to_client")

	suite.statsd.EventTraffic(
		mtglib.NewEventTraffic("connID", 90, false))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(),
		"mtg.telegram_traffic:90|c|#telegram_ip:10.1.0.10,dc:2,direction:from_client")

	suite.statsd.EventFinish(mtglib.NewEventFinish("connID"))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(),
		"mtg.telegram_connections:-1|g|#telegram_ip:10.1.0.10,dc:2")
	suite.Contains(suite.statsdServer.String(),
		"mtg.client_connections:-1|g|#ip_family:ipv4")

	suite.NotContains(suite.statsdServer.String(), "domain_fronting_traffic")
	suite.NotContains(suite.statsdServer.String(), "domain_fronting_connections")
}

func (suite *StatsdTestSuite) TestDomainFrontingPath() {
	suite.statsd.EventStart(
		mtglib.NewEventStart("connID", net.ParseIP("10.0.0.10")))
	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.client_connections:+1|g|#ip_family:ipv4", suite.statsdServer.String())

	suite.statsd.EventDomainFronting(mtglib.NewEventDomainFronting("connID"))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(), "mtg.domain_fronting:1|c")
	suite.Contains(suite.statsdServer.String(),
		`mtg.domain_fronting_connections:+1|g|#ip_family:ipv4`)

	suite.statsd.EventTraffic(
		mtglib.NewEventTraffic("connID", 30, true))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(),
		`mtg.domain_fronting_traffic:30|c|#direction:to_client`)

	suite.statsd.EventTraffic(
		mtglib.NewEventTraffic("connID", 90, false))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(),
		`mtg.domain_fronting_traffic:90|c|#direction:from_client`)

	suite.statsd.EventFinish(mtglib.NewEventFinish("connID"))
	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(),
		"mtg.domain_fronting_connections:-1|g|#ip_family:ipv4")
	suite.Contains(suite.statsdServer.String(),
		"mtg.client_connections:-1|g|#ip_family:ipv4")

	suite.NotContains(suite.statsdServer.String(), "telegram_traffic")
	suite.NotContains(suite.statsdServer.String(), "telegram_connections")
}

func (suite *StatsdTestSuite) TestEventConcurrencyLimited() {
	suite.statsd.EventConcurrencyLimited(mtglib.NewEventConcurrencyLimited())

	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.concurrency_limited:1|c", suite.statsdServer.String())
}

func (suite *StatsdTestSuite) TestEventIPBlocklisted() {
	suite.statsd.EventIPBlocklisted(
		mtglib.NewEventIPBlocklisted(net.ParseIP("10.0.0.10")))

	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.ip_blocklisted:1|c|#ip_list:blocklist", suite.statsdServer.String())
}

func (suite *StatsdTestSuite) TestEventIPAllowlisted() {
	suite.statsd.EventIPBlocklisted(
		mtglib.NewEventIPAllowlisted(net.ParseIP("10.0.0.10")))

	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.ip_blocklisted:1|c|#ip_list:allowlist", suite.statsdServer.String())
}

func (suite *StatsdTestSuite) TestEventReplayAttack() {
	suite.statsd.EventReplayAttack(mtglib.NewEventReplayAttack("connID"))

	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.replay_attacks:1|c", suite.statsdServer.String())
}

func (suite *StatsdTestSuite) TestEventIPListSizeAllowlist() {
	suite.statsd.EventIPListSize(mtglib.NewEventIPListSize(10, false))

	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(), "mtg.iplist_size:10|g")
	suite.Contains(suite.statsdServer.String(), "allowlist")
}

func (suite *StatsdTestSuite) TestEventIPListSizeBlocklist() {
	suite.statsd.EventIPListSize(mtglib.NewEventIPListSize(10, true))

	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(), "mtg.iplist_size:10|g")
	suite.Contains(suite.statsdServer.String(), "blocklist")
}

func TestStatsd(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StatsdTestSuite{})
}
