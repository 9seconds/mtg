package stats_test

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/stats"
	statsd "github.com/smira/go-statsd"
	"github.com/stretchr/testify/suite"
)

const statsdSleepTime = 3 * statsd.DefaultFlushInterval

type statsdFakeServer struct {
	conn *net.UDPConn
	buf  *bytes.Buffer
}

func (s statsdFakeServer) Addr() string {
	return s.conn.LocalAddr().String()
}

func (s statsdFakeServer) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}

	return nil
}

func (s statsdFakeServer) String() string {
	return strings.TrimSpace(s.buf.String())
}

func statsdNewFakeServer() statsdFakeServer {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: 0,
	})
	if err != nil {
		panic(err)
	}

	buf := &bytes.Buffer{}

	go func() {
		currentBuffer := make([]byte, 4096)

		for {
			n, _, err := conn.ReadFromUDP(currentBuffer)
			if n > 0 {
				buf.Write(currentBuffer[:n])
			}

			if err != nil {
				return
			}
		}
	}()

	return statsdFakeServer{
		conn: conn,
		buf:  buf,
	}
}

type StatsdTestSuite struct {
	suite.Suite

	statsdServer statsdFakeServer
	factory      stats.StatsdFactory
	statsd       events.Observer
}

func (suite *StatsdTestSuite) SetupTest() {
	suite.statsdServer = statsdNewFakeServer()

	factory, err := stats.NewStatsd(suite.statsdServer.Addr(), "mtg.", "datadog")
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

func (suite *StatsdTestSuite) TestEventStartFinish() {
	suite.statsd.EventStart(mtglib.EventStart{
		CreatedAt: time.Now(),
		ConnID:    "connID",
		RemoteIP:  net.ParseIP("10.0.0.10"),
	})

	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.active_connections:+1|g|#ip_type:ipv4", suite.statsdServer.String())

	suite.statsd.EventFinish(mtglib.EventFinish{
		CreatedAt: time.Now(),
		ConnID:    "connID",
	})

	time.Sleep(statsdSleepTime)
	suite.Contains(suite.statsdServer.String(), "mtg.session_duration")
}

func (suite *StatsdTestSuite) TestEventConcurrencyLimited() {
	suite.statsd.EventConcurrencyLimited(mtglib.EventConcurrencyLimited{
		CreatedAt: time.Now(),
	})

	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.concurrency_limited:1|c", suite.statsdServer.String())
}

func (suite *StatsdTestSuite) TestEventIPBlocklisted() {
	suite.statsd.EventIPBlocklisted(mtglib.EventIPBlocklisted{
		CreatedAt: time.Now(),
		RemoteIP:  net.ParseIP("10.0.0.10"),
	})

	time.Sleep(statsdSleepTime)
	suite.Equal("mtg.ip_blocklisted:1|c|#ip_type:ipv4", suite.statsdServer.String())
}

func TestStatsd(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StatsdTestSuite{})
}
