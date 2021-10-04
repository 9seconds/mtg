package mtglib

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type NoopLogger struct{}

func (n NoopLogger) Named(_ string) Logger             { return n }
func (n NoopLogger) BindInt(_ string, _ int) Logger    { return n }
func (n NoopLogger) BindStr(_, _ string) Logger        { return n }
func (n NoopLogger) BindJSON(_, _ string) Logger       { return n }
func (n NoopLogger) Printf(_ string, _ ...interface{}) {}
func (n NoopLogger) Info(_ string)                     {}
func (n NoopLogger) Warning(_ string)                  {}
func (n NoopLogger) Debug(_ string)                    {}
func (n NoopLogger) InfoError(_ string, _ error)       {}
func (n NoopLogger) WarningError(_ string, _ error)    {}
func (n NoopLogger) DebugError(_ string, _ error)      {}

type EventStreamMock struct {
	mock.Mock
}

func (e *EventStreamMock) Send(ctx context.Context, evt Event) {
	e.Called(ctx, evt)
}
