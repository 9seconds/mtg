package antireplay_test

import (
	"testing"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/stretchr/testify/suite"
)

type NoopTestSuite struct {
	suite.Suite
}

func (suite *NoopTestSuite) TestOp() {
	filter := antireplay.NewNoop()

	suite.False(filter.SeenBefore([]byte{1, 2, 3}))
	suite.False(filter.SeenBefore([]byte{4, 5, 6}))
	suite.False(filter.SeenBefore([]byte{1, 2, 3}))
	suite.False(filter.SeenBefore([]byte{4, 5, 6}))
}

func TestNoop(t *testing.T) {
	t.Parallel()
	suite.Run(t, &NoopTestSuite{})
}
