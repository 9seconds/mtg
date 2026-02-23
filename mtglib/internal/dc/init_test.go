package dc

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type LoggerMock struct {
	mock.Mock
}

func (m *LoggerMock) Info(msg string) {
	m.Called(msg)
}

func (m *LoggerMock) WarningError(msg string, err error) {
	m.Called(msg, err)
}

type UpdaterTestSuiteBase struct {
	suite.Suite

	ctx        context.Context
	ctxCancel  context.CancelFunc
	loggerMock *LoggerMock
}

func (s *UpdaterTestSuiteBase) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())

	s.loggerMock = &LoggerMock{}
	s.loggerMock.On("Info", mock.AnythingOfType("string"))
	s.loggerMock.On("WarningError", mock.AnythingOfType("string"), mock.Anything)

	s.ctx = ctx
	s.ctxCancel = cancel
}

func (s *UpdaterTestSuiteBase) TearDownTest() {
	s.ctxCancel()
}
