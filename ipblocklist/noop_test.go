package ipblocklist_test

import (
	"net"
	"testing"

	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/stretchr/testify/suite"
)

type NoopTestSuite struct {
	suite.Suite
}

func (suite *NoopTestSuite) TestOp() {
	suite.False(ipblocklist.NewNoop().Contains(net.ParseIP("10.0.0.10")))
	suite.False(ipblocklist.NewNoop().Contains(net.ParseIP("10.0.0.10")))
}

func (suite *NoopTestSuite) TestRun() {
	blocklist := ipblocklist.NewNoop()

	blocklist.Run(0)
	blocklist.Shutdown()
}

func TestNoop(t *testing.T) {
	t.Parallel()
	suite.Run(t, &NoopTestSuite{})
}
