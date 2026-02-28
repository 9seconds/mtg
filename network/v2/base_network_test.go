package network_test

import (
	"context"
	"testing"

	"github.com/9seconds/mtg/v2/network/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type BaseNetworkTestSuite struct {
	EchoServerTestSuite

	net network.Network
}

func (suite *BaseNetworkTestSuite) SetupSuite() {
	suite.EchoServerTestSuite.SetupSuite()

	suite.net = network.New(nil, "agent", 0, 0, 0)
}

func (suite *BaseNetworkTestSuite) TestDialUnknownNetwork() {
	testData := []string{
		"udp",
		"udp4",
		"udp6",
		"unix",
	}

	for _, name := range testData {
		suite.T().Run(name, func(t *testing.T) {
			_, err := suite.net.Dial(name, suite.EchoServerAddr())
			assert.Error(t, err)
		})
	}
}

func (suite *BaseNetworkTestSuite) TestDial() {
	conn, err := suite.net.Dial("tcp4", suite.EchoServerAddr())
	suite.NoError(err)

	buf := []byte{1, 2, 3, 4, 5}
	n, err := conn.Write(buf)
	suite.Equal(5, n)
	suite.NoError(err)

	another := make([]byte, len(buf))
	n, err = conn.Read(another)
	suite.NoError(err)
	suite.Equal(len(another), n)
	suite.Equal(buf, another)
}

func (suite *BaseNetworkTestSuite) TestDialContextOk() {
	conn, err := suite.net.DialContext(context.Background(), "tcp4", suite.EchoServerAddr())
	suite.NoError(err)

	buf := []byte{1, 2, 3, 4, 5}
	n, err := conn.Write(buf)
	suite.Equal(5, n)
	suite.NoError(err)

	another := make([]byte, len(buf))
	n, err = conn.Read(another)
	suite.NoError(err)
	suite.Equal(len(another), n)
	suite.Equal(buf, another)
}

func (suite *BaseNetworkTestSuite) TestDialContextClosed() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := suite.net.DialContext(ctx, "tcp4", suite.EchoServerAddr())
	suite.ErrorIs(err, ctx.Err())
}

func TestNetworkBase(t *testing.T) {
	t.Parallel()
	suite.Run(t, &BaseNetworkTestSuite{})
}
