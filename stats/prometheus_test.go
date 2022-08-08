package stats_test

import (
	"fmt"
	"io"
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

	resp, err := http.Get(addr) //nolint: noctx
	if err != nil {
		return "", err //nolint: wrapcheck
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err //nolint: wrapcheck
	}

	return string(data), nil
}

func (suite *PrometheusTestSuite) SetupTest() {
	suite.httpListener, _ = net.Listen("tcp", "127.0.0.1:0")
	suite.factory = stats.NewPrometheus("mtg", "/")
	suite.prometheus = suite.factory.Make()

	go suite.factory.Serve(suite.httpListener) //nolint: errcheck
}

func (suite *PrometheusTestSuite) TearDownTest() {
	suite.prometheus.Shutdown()
	suite.NoError(suite.factory.Close())
	suite.httpListener.Close()
}

func (suite *PrometheusTestSuite) TestTelegramPath() {
	suite.prometheus.EventStart(
		mtglib.NewEventStart("connID", net.ParseIP("10.0.0.10")))
	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_client_connections{ip_family="ipv4"} 1`)

	suite.prometheus.EventConnectedToDC(
		mtglib.NewEventConnectedToDC("connID", net.ParseIP("10.0.0.1"), 4))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_telegram_connections{dc="4",telegram_ip="10.0.0.1"} 1`)

	suite.prometheus.EventTraffic(
		mtglib.NewEventTraffic("connID", 200, true))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_telegram_traffic{dc="4",direction="to_client",telegram_ip="10.0.0.1"} 200`)

	suite.prometheus.EventTraffic(
		mtglib.NewEventTraffic("connID", 100, false))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_telegram_traffic{dc="4",direction="from_client",telegram_ip="10.0.0.1"} 100`)

	suite.prometheus.EventFinish(mtglib.NewEventFinish("connID"))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_client_connections{ip_family="ipv4"} 0`)
	suite.Contains(data, `mtg_telegram_connections{dc="4",telegram_ip="10.0.0.1"} 0`)
}

func (suite *PrometheusTestSuite) TestDomainFrontingPath() {
	suite.prometheus.EventStart(
		mtglib.NewEventStart("connID", net.ParseIP("10.0.0.10")))
	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_client_connections{ip_family="ipv4"} 1`)

	suite.prometheus.EventDomainFronting(mtglib.NewEventDomainFronting("connID"))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_domain_fronting 1`)
	suite.Contains(data, `mtg_domain_fronting_connections{ip_family="ipv4"} 1`)

	suite.prometheus.EventTraffic(
		mtglib.NewEventTraffic("connID", 200, true))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_domain_fronting_traffic{direction="to_client"} 200`)

	suite.prometheus.EventTraffic(
		mtglib.NewEventTraffic("connID", 100, false))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_domain_fronting_traffic{direction="from_client"} 100`)

	suite.prometheus.EventFinish(mtglib.NewEventFinish("connID"))
	time.Sleep(100 * time.Millisecond)

	data, err = suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_client_connections{ip_family="ipv4"} 0`)
	suite.Contains(data, `mtg_domain_fronting_connections{ip_family="ipv4"} 0`)
}

func (suite *PrometheusTestSuite) TestEventConcurrencyLimited() {
	suite.prometheus.EventConcurrencyLimited(mtglib.NewEventConcurrencyLimited())

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_concurrency_limited 1`)
}

func (suite *PrometheusTestSuite) TestEventIPBlocklisted() {
	suite.prometheus.EventIPBlocklisted(
		mtglib.NewEventIPBlocklisted(net.ParseIP("2001:db8::68")))

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_ip_blocklisted{ip_list="blocklist"} 1`)
}

func (suite *PrometheusTestSuite) TestEventIPAllowlisted() {
	suite.prometheus.EventIPBlocklisted(
		mtglib.NewEventIPAllowlisted(net.ParseIP("2001:db8::68")))

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_ip_blocklisted{ip_list="allowlist"} 1`)
}

func (suite *PrometheusTestSuite) TestEventReplayAttack() {
	suite.prometheus.EventReplayAttack(mtglib.NewEventReplayAttack("connID"))

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_replay_attacks 1`)
}

func (suite *PrometheusTestSuite) TestEventIPListSize() {
	suite.prometheus.EventIPListSize(mtglib.NewEventIPListSize(10, false))
	suite.prometheus.EventIPListSize(mtglib.NewEventIPListSize(3, true))

	time.Sleep(100 * time.Millisecond)

	data, err := suite.Get()
	suite.NoError(err)
	suite.Contains(data, `mtg_iplist_size{ip_list="allowlist"} 10`)
	suite.Contains(data, `mtg_iplist_size{ip_list="blocklist"} 3`)
}

func TestPrometheus(t *testing.T) {
	t.Parallel()
	suite.Run(t, &PrometheusTestSuite{})
}
