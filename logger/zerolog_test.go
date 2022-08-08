package logger_test

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type zeroLoggerLogMessage struct {
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"`
	StrParam  string `json:"strparam"`
	IntParam  int    `json:"intparam"`
	Logger    string `json:"logger"`
	Error     string `json:"error"`
	Message   string `json:"message"`
}

type ZeroLoggerTestSuite struct {
	suite.Suite
}

func (suite *ZeroLoggerTestSuite) SetupSuite() {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.TimestampFieldName = "timestamp"
	zerolog.LevelFieldName = "level"
}

func (suite *ZeroLoggerTestSuite) TestLog() {
	testData := map[string]func(mtglib.Logger){
		"info":        func(l mtglib.Logger) { l.Info("hello") },
		"warn":        func(l mtglib.Logger) { l.Warning("hello") },
		"printf":      func(l mtglib.Logger) { l.Printf("hello") },
		"debug":       func(l mtglib.Logger) { l.Debug("hello") },
		"info-error":  func(l mtglib.Logger) { l.InfoError("hello", io.EOF) },
		"warn-error":  func(l mtglib.Logger) { l.WarningError("hello", io.EOF) },
		"debug-error": func(l mtglib.Logger) { l.DebugError("hello", io.EOF) },
	}

	for k, v := range testData {
		name := k
		callback := v
		level := strings.TrimSuffix(name, "-error")

		suite.T().Run(name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			log := logger.NewZeroLogger(zerolog.New(buf).With().Timestamp().Logger())

			callback(log.Named("name").BindInt("intparam", 1).BindStr("strparam", name))

			msg := &zeroLoggerLogMessage{}
			assert.NoError(t, json.Unmarshal(buf.Bytes(), msg))

			timestamp := time.Unix(msg.Timestamp/1000, (msg.Timestamp%1000)*1_000_000)
			assert.WithinDuration(t, time.Now(), timestamp, 100*time.Millisecond)

			if level == "printf" {
				level = "debug"
			}

			assert.Equal(t, level, msg.Level)
			assert.Equal(t, name, msg.StrParam)
			assert.EqualValues(t, 1, msg.IntParam)
			assert.Equal(t, "name", msg.Logger)
			assert.Equal(t, "hello", msg.Message)

			if level != name && name != "printf" {
				assert.Equal(t, io.EOF.Error(), msg.Error)
			} else {
				assert.Empty(t, msg.Error)
			}
		})
	}
}

func (suite *ZeroLoggerTestSuite) TestIndependence() {
	buf := &bytes.Buffer{}
	log := logger.NewZeroLogger(zerolog.New(buf).With().Timestamp().Logger())

	log1 := log.Named("1")
	log2 := log.Named("2")
	log12 := log1.Named("2")

	log1.BindInt("param", 1).Info("hello")

	log1Output := buf.String()

	buf.Reset()

	log2.BindInt("lalala", 2).Info("hello")

	log2Output := buf.String()

	buf.Reset()

	log12.BindStr("tttt", "qqq").Info("hello")

	log12Output := buf.String()

	suite.NotContains("lalala", log1Output)
	suite.NotContains("tttt", log1Output)
	suite.NotContains("param", log2Output)
	suite.NotContains("tttt", log1Output)
	suite.NotContains("param", log12Output)
	suite.NotContains("lalala", log12Output)
}

func TestZeroLogger(t *testing.T) { //nolint: paralleltest
	suite.Run(t, &ZeroLoggerTestSuite{})
}
