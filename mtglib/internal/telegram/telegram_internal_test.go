package telegram

import (
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TelegramTestSuite struct {
	suite.Suite

	dialerMock *testlib.MtglibNetworkMock
	t          *Telegram
}

func (suite *TelegramTestSuite) SetupTest() {
	suite.dialerMock = &testlib.MtglibNetworkMock{}
	suite.t, _ = New(suite.dialerMock, "prefer-ipv4", false)
}

func (suite *TelegramTestSuite) TearDownTest() {
	suite.dialerMock.AssertExpectations(suite.T())
}

func (suite *TelegramTestSuite) TestUnknownDC() {
	testData := []int{
		-1,
		0,
		6,
		100,
	}

	for _, v := range testData {
		value := v

		suite.T().Run(strconv.Itoa(value), func(t *testing.T) {
			_, err := suite.t.Dial(context.Background(), value)
			assert.Error(t, err)
			assert.False(t, suite.t.IsKnownDC(value))
		})
	}
}

func (suite *TelegramTestSuite) TestDialToCorrectIPs() {
	testData := map[int][]tgAddr{}

	for i := 1; i <= 5; i++ {
		testData[i] = []tgAddr{}
		testData[i] = append(testData[i], productionV4Addresses[i-1]...)
		testData[i] = append(testData[i], productionV6Addresses[i-1]...)
	}

	for i, v := range testData {
		idx := i
		addresses := v

		suite.T().Run(strconv.Itoa(idx), func(t *testing.T) {
			for _, addr := range addresses {
				suite.dialerMock.
					On("DialContext", mock.Anything, addr.network, addr.address).
					Once().
					Return((*net.TCPConn)(nil), io.EOF)
			}

			_, err := suite.t.Dial(context.Background(), idx)
			assert.True(t, errors.Is(err, io.EOF))
			assert.True(t, suite.t.IsKnownDC(idx))
		})
	}
}

func (suite *TelegramTestSuite) TestDialPreferIPRange() {
	testData := map[string][]tgAddr{
		"prefer-ipv4": {testV4Addresses[0][0], testV6Addresses[0][0]},
		"prefer-ipv6": {testV6Addresses[0][0], testV4Addresses[0][0]},
		"only-ipv4":   {testV4Addresses[0][0]},
		"only-ipv6":   {testV6Addresses[0][0]},
	}

	for k, v := range testData {
		name := k
		addresses := v

		suite.T().Run(name, func(t *testing.T) {
			for _, addr := range addresses {
				suite.dialerMock.
					On("DialContext", mock.Anything, addr.network, addr.address).
					Once().
					Return((*net.TCPConn)(nil), io.EOF)
			}

			tg, _ := New(suite.dialerMock, name, true)
			_, err := tg.Dial(context.Background(), 1)

			assert.True(t, errors.Is(err, io.EOF))
		})
	}
}

func (suite *TelegramTestSuite) TestDialPreferIPPriority() {
	testData := map[string]tgAddr{
		"prefer-ipv4": productionV4Addresses[0][0],
		"prefer-ipv6": productionV6Addresses[0][0],
	}

	for k, v := range testData {
		name := k
		addr := v

		suite.T().Run(name, func(t *testing.T) {
			conn := &net.TCPConn{}

			suite.dialerMock.
				On("DialContext", mock.Anything, addr.network, addr.address).
				Once().
				Return(conn, nil)

			tg, _ := New(suite.dialerMock, name, false)

			res, err := tg.Dial(context.Background(), 1)
			assert.NoError(t, err)
			assert.Equal(t, conn, res)
		})
	}
}

func (suite *TelegramTestSuite) TestUnknownPreferIP() {
	_, err := New(suite.dialerMock, "xxx", false)
	suite.Error(err)
}

func (suite *TelegramTestSuite) TestFallbackDC() {
	dcs := make([]int, 10)

	for i := 0; i < len(dcs); i++ {
		dcs[i] = suite.t.GetFallbackDC()
	}

	for _, v := range dcs {
		value := v

		suite.T().Run(strconv.Itoa(value), func(t *testing.T) {
			assert.True(t, suite.t.IsKnownDC(value))
		})
	}
}

func TestTelegram(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TelegramTestSuite{})
}
