package timeattack_test

import (
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/timeattack"
	"github.com/stretchr/testify/suite"
)

type NoopTestSuite struct {
	suite.Suite
}

func (suite *NoopTestSuite) TestOp() {
	d := timeattack.NewNoop()

	suite.NoError(d.Valid(time.Now()))
	suite.NoError(d.Valid(time.Now().Add(time.Hour)))
	suite.NoError(d.Valid(time.Now().Add(-time.Hour)))
}

func TestNoop(t *testing.T) {
	t.Parallel()
	suite.Run(t, &NoopTestSuite{})
}
