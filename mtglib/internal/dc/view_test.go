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
		overrides: dcAddrSet{
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
		collected: dcAddrSet{
			v4: map[int][]Addr{
				1: {
					{Network: "tcp4", Address: "127.1.0.1:443"},
				},
			},
		},
	}
}

func (suite *ViewTestSuite) TestGetV4() {
	testData := map[int][]Addr{
		111: {
			{"tcp4", "127.0.0.1:443"},
		},
		203: {
			{"tcp4", "127.0.0.2:443"},
			{"tcp4", "91.105.192.100:443"},
		},
		2: {
			{"tcp4", "149.154.167.51:443"},
			{"tcp4", "95.161.76.100:443"},
		},
		1: {
			{"tcp4", "127.1.0.1:443"},
			{"tcp4", "149.154.175.50:443"},
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
			{"tcp6", "xxx"},
			{"tcp6", "[2a0a:f280:0203:000a:5000:0000:0000:0100]:443"},
		},
		1: {
			{"tcp6", "[2001:b28:f23d:f001::a]:443"},
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
