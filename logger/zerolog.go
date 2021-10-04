package logger

import (
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/rs/zerolog"
)

const loggerFieldName = "logger"

type zeroLogContextVarType uint8

const (
	zeroLogContextVarTypeUnknown zeroLogContextVarType = iota
	zeroLogContextVarTypeStr
	zeroLogContextVarTypeInt
	zeroLogContextVarTypeJSON
)

type zeroLogContext struct {
	name string
	log  *zerolog.Logger

	ctxVarType zeroLogContextVarType
	ctxVarName string
	ctxVarStr  string
	ctxVarInt  int

	parent *zeroLogContext
}

func (z *zeroLogContext) Named(name string) mtglib.Logger {
	loggerName := z.name
	if loggerName == "" {
		loggerName = name
	} else {
		loggerName += "." + name
	}

	return &zeroLogContext{
		name:   loggerName,
		log:    z.log,
		parent: z,
	}
}

func (z *zeroLogContext) BindInt(name string, value int) mtglib.Logger {
	return &zeroLogContext{
		name:       z.name,
		log:        z.log,
		ctxVarType: zeroLogContextVarTypeInt,
		ctxVarInt:  value,
		ctxVarName: name,
		parent:     z,
	}
}

func (z *zeroLogContext) BindStr(name, value string) mtglib.Logger {
	return &zeroLogContext{
		name:       z.name,
		log:        z.log,
		ctxVarType: zeroLogContextVarTypeStr,
		ctxVarStr:  value,
		ctxVarName: name,
		parent:     z,
	}
}

func (z *zeroLogContext) BindJSON(name, value string) mtglib.Logger {
	return &zeroLogContext{
		name:       z.name,
		log:        z.log,
		ctxVarType: zeroLogContextVarTypeJSON,
		ctxVarName: name,
		ctxVarStr:  value,
		parent:     z,
	}
}

func (z *zeroLogContext) Printf(format string, args ...interface{}) {
	z.Debug(fmt.Sprintf(format, args...))
}

func (z *zeroLogContext) Info(msg string) {
	z.InfoError(msg, nil)
}

func (z *zeroLogContext) Warning(msg string) {
	z.WarningError(msg, nil)
}

func (z *zeroLogContext) Debug(msg string) {
	z.DebugError(msg, nil)
}

func (z *zeroLogContext) InfoError(msg string, err error) {
	z.emitLog(z.log.Info(), msg, err)
}

func (z *zeroLogContext) WarningError(msg string, err error) {
	z.emitLog(z.log.Warn(), msg, err)
}

func (z *zeroLogContext) DebugError(msg string, err error) {
	z.emitLog(z.log.Debug(), msg, err)
}

func (z *zeroLogContext) emitLog(evt *zerolog.Event, msg string, err error) {
	z.attachCtx(evt)

	for current := z.parent; current != nil; current = current.parent {
		current.attachCtx(evt)
	}

	evt.Str(loggerFieldName, z.name).Err(err).Msg(msg)
}

func (z *zeroLogContext) attachCtx(evt *zerolog.Event) {
	switch z.ctxVarType {
	case zeroLogContextVarTypeStr:
		evt.Str(z.ctxVarName, z.ctxVarStr)
	case zeroLogContextVarTypeInt:
		evt.Int(z.ctxVarName, z.ctxVarInt)
	case zeroLogContextVarTypeJSON:
		evt.RawJSON(z.ctxVarName, []byte(z.ctxVarStr))
	case zeroLogContextVarTypeUnknown:
	}
}

// NewZeroLogger returns a logger which is using rs/zerolog library.
func NewZeroLogger(log zerolog.Logger) mtglib.Logger {
	return &zeroLogContext{
		log: &log,
	}
}
