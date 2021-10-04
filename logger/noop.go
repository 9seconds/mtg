package logger

import "github.com/9seconds/mtg/v2/mtglib"

type noopLogger struct{}

func (n noopLogger) Named(_ string) mtglib.Logger          { return n }
func (n noopLogger) BindInt(_ string, _ int) mtglib.Logger { return n }
func (n noopLogger) BindStr(_, _ string) mtglib.Logger     { return n }
func (n noopLogger) BindJSON(_, _ string) mtglib.Logger    { return n }
func (n noopLogger) Printf(_ string, _ ...interface{})     {}
func (n noopLogger) Info(_ string)                         {}
func (n noopLogger) Warning(_ string)                      {}
func (n noopLogger) Debug(_ string)                        {}
func (n noopLogger) InfoError(_ string, _ error)           {}
func (n noopLogger) WarningError(_ string, _ error)        {}
func (n noopLogger) DebugError(_ string, _ error)          {}

// NewNoopLogger returns a logger which discards all events.
func NewNoopLogger() mtglib.Logger {
	return noopLogger{}
}
