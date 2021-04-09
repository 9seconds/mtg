package logger_test

import (
	"io"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/stretchr/testify/suite"
)

type NoopLoggerTestSuite struct {
	suite.Suite
}

func (suite *NoopLoggerTestSuite) TestLog() {
	suite.Empty(testlib.CaptureStdout(func() {
		suite.Empty(testlib.CaptureStderr(func() {
			log := logger.NewNoopLogger().Named("name")

			log.BindInt("int", 1).BindStr("str", "1").Printf("info", 1, 2)
			log.BindInt("int", 1).BindStr("str", "1").Info("info")
			log.BindInt("int", 1).BindStr("str", "1").Warning("info")
			log.BindInt("int", 1).BindStr("str", "1").Debug("info")
			log.BindInt("int", 1).BindStr("str", "1").InfoError("info", io.EOF)
			log.BindInt("int", 1).BindStr("str", "1").WarningError("info", io.EOF)
			log.BindInt("int", 1).BindStr("str", "1").DebugError("info", io.EOF)
		}))
	}))
}

func TestNoopLogger(t *testing.T) {
	t.Parallel()
	suite.Run(t, &NoopLoggerTestSuite{})
}
