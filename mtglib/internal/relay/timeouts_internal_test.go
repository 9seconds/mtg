package relay

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TimeoutsTestSuite struct {
	suite.Suite
}

func (suite *TimeoutsTestSuite) TestGetConnectionTimeToLive() {
	for i := 0; i < 100; i++ {
		value := getConnectionTimeToLive()
		message := fmt.Sprintf("generated value is %v", value)

		suite.GreaterOrEqual(value, ConnectionTimeToLiveMin, message)
		suite.LessOrEqual(value, ConnectionTimeToLiveMax, message)
	}
}

func (suite *TimeoutsTestSuite) TestGetTimeout() {
	for i := 0; i < 100; i++ {
		value := getTimeout()
		message := fmt.Sprintf("generated value is %v", value)

		suite.GreaterOrEqual(value, TimeoutMin, message)
		suite.LessOrEqual(value, TimeoutMax, message)
	}
}

func TestTimeouts(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TimeoutsTestSuite{})
}
