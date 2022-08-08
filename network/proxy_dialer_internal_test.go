package network

import (
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ProxyDialerTestSuite struct {
	suite.Suite

	u *url.URL
}

func (suite *ProxyDialerTestSuite) SetupSuite() {
	u, _ := url.Parse("socks5://hello:world@10.0.0.10:3128")
	suite.u = u
}

func (suite *ProxyDialerTestSuite) TestSetupDefaults() {
	d := newProxyDialer(&DialerMock{}, suite.u).(*circuitBreakerDialer) //nolint: forcetypeassert
	suite.EqualValues(ProxyDialerOpenThreshold, d.openThreshold)
	suite.EqualValues(ProxyDialerHalfOpenTimeout, d.halfOpenTimeout)
	suite.EqualValues(ProxyDialerResetFailuresTimeout, d.resetFailuresTimeout)
}

func (suite *ProxyDialerTestSuite) TestSetupValuesAllOk() {
	query := url.Values{}
	query.Set("open_threshold", "30")
	query.Set("reset_failures_timeout", "1s")
	query.Set("half_open_timeout", "2s")
	suite.u.RawQuery = query.Encode()

	d := newProxyDialer(&DialerMock{}, suite.u).(*circuitBreakerDialer) //nolint: forcetypeassert
	suite.EqualValues(30, d.openThreshold)
	suite.EqualValues(2*time.Second, d.halfOpenTimeout)
	suite.EqualValues(time.Second, d.resetFailuresTimeout)
}

func (suite *ProxyDialerTestSuite) TestOpenThreshold() {
	query := url.Values{}
	params := []string{"-30", "aaa", "1.0", "-1.0"}

	for _, v := range params {
		param := v
		suite.T().Run(v, func(t *testing.T) {
			query.Set("open_threshold", param)
			suite.u.RawQuery = query.Encode()

			d := newProxyDialer(&DialerMock{}, suite.u).(*circuitBreakerDialer) //nolint: forcetypeassert
			assert.EqualValues(t, ProxyDialerOpenThreshold, d.openThreshold)
		})
	}
}

func (suite *ProxyDialerTestSuite) TestHalfOpenTimeout() {
	query := url.Values{}
	params := []string{"-30", "30", "aaa", "-3.0", "3.0"}

	for _, v := range params {
		param := v
		suite.T().Run(v, func(t *testing.T) {
			query.Set("half_open_timeout", param)
			suite.u.RawQuery = query.Encode()

			d := newProxyDialer(&DialerMock{}, suite.u).(*circuitBreakerDialer) //nolint: forcetypeassert
			assert.EqualValues(t, ProxyDialerHalfOpenTimeout, d.halfOpenTimeout)
		})
	}
}

func (suite *ProxyDialerTestSuite) TestResetFailuresTimeout() {
	query := url.Values{}
	params := []string{"-30", "30", "aaa", "-3.0", "3.0"}

	for _, v := range params {
		param := v
		suite.T().Run(v, func(t *testing.T) {
			query.Set("reset_failures_timeout", param)
			suite.u.RawQuery = query.Encode()

			d := newProxyDialer(&DialerMock{}, suite.u).(*circuitBreakerDialer) //nolint: forcetypeassert
			assert.EqualValues(t, ProxyDialerHalfOpenTimeout, d.halfOpenTimeout)
		})
	}
}

func TestProxyDialer(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProxyDialerTestSuite{})
}
