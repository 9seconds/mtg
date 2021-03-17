package stats_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/stats"
	"github.com/stretchr/testify/suite"
)

type PrometheusTestSuite struct {
	suite.Suite

	httpListener net.Listener
	factory      *stats.PrometheusFactory
	prometheus   events.Observer
}

func (suite *PrometheusTestSuite) Get() (string, error) {
	addr := fmt.Sprintf("http://%s/", suite.httpListener.Addr().String())

	resp, err := http.Get(addr) // nolint: noctx
	if err != nil {
		return "", err // nolint: wrapcheck
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err // nolint: wrapcheck
	}

	return string(data), nil
}

func (suite *PrometheusTestSuite) SetupTest() {
	suite.httpListener, _ = net.Listen("tcp", "127.0.0.1:0")
	suite.factory = stats.NewPrometheus("mtg", "/")
	suite.prometheus = suite.factory.Make()

	go suite.factory.Serve(suite.httpListener) // nolint: errcheck
}

func (suite *PrometheusTestSuite) TearDownTest() {
	suite.prometheus.Shutdown()
	suite.NoError(suite.factory.Close())
	suite.httpListener.Close()
}

func (suite *PrometheusTestSuite) TestEventStartFinish() {
	suite.prometheus.EventStart(mtglib.EventStart{
		CreatedAt: time.Now(),
		ConnID:    "connID",
		RemoteIP:  net.ParseIP("10.0.0.10"),
	})

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_active_connections{ip_type="ipv4"} 1`)

	suite.prometheus.EventFinish(mtglib.EventFinish{
		CreatedAt: time.Now(),
		ConnID:    "connID",
	})

	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_active_connections{ip_type="ipv4"} 0`)
}

func (suite *PrometheusTestSuite) TestEventConcurrencyLimited() {
	suite.prometheus.EventConcurrencyLimited(mtglib.EventConcurrencyLimited{
		CreatedAt: time.Now(),
	})

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_concurrency_limited 1`)
}

func (suite *PrometheusTestSuite) TestEventIPBlocklisted() {
	suite.prometheus.EventIPBlocklisted(mtglib.EventIPBlocklisted{
		CreatedAt: time.Now(),
		RemoteIP:  net.ParseIP("2001:db8::68"),
	})

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_ip_blocklisted{ip_type="ipv6"} 1`)
}

func TestPrometheus(t *testing.T) {
	t.Parallel()
	suite.Run(t, &PrometheusTestSuite{})
}
