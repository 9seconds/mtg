package dc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ViewTestSuite struct {
	suite.Suite

	view dcView
}

func (suite *ViewTestSuite) SetupSuite() {
	suite.view = dcView{
		publicConfigs: dcAddrSet{
			v4: map[int][]Addr{
				111: {
					{Network: "tcp4", Address: "127.0.0.1:443"},
				},
				203: {
					{Network: "tcp4", Address: "127.0.0.2:443"},
				},
			},
			v6: map[int][]Addr{
				203: {
					{Network: "tcp6", Address: "xxx"},
				},
			},
		},
	}
}

func (suite *ViewTestSuite) TestGetV4() {
	testData := map[int][]Addr{
		111: {
			{Network: "tcp4", Address: "127.0.0.1:443"},
		},
		203: {
			{Network: "tcp4", Address: "127.0.0.2:443"},
			{Network: "tcp4", Address: "91.105.192.100:443"},
		},
		2: {
			{Network: "tcp4", Address: "149.154.167.51:443"},
			{Network: "tcp4", Address: "95.161.76.100:443"},
		},
	}

	for dc, addresses := range testData {
		suite.T().Run(fmt.Sprintf("dc%d", dc), func(t *testing.T) {
			assert.ElementsMatch(t, addresses, suite.view.getV4(dc))
		})
	}
}

func (suite *ViewTestSuite) TestGetV6() {
	testData := map[int][]Addr{
		111: {},
		203: {
			{Network: "tcp6", Address: "xxx"},
			{Network: "tcp6", Address: "[2a0a:f280:0203:000a:5000:0000:0000:0100]:443"},
		},
		1: {
			{Network: "tcp6", Address: "[2001:b28:f23d:f001::a]:443"},
		},
	}

	for dc, addresses := range testData {
		suite.T().Run(fmt.Sprintf("dc%d", dc), func(t *testing.T) {
			assert.ElementsMatch(t, addresses, suite.view.getV6(dc))
		})
	}
}

func TestView(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ViewTestSuite{})
}
