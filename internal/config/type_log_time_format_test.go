package config_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeLogTimeFormatTestStruct struct {
	Value config.TypeLogTimeFormat `json:"value"`
}

type TypeLogTimeFormatTestSuite struct {
	suite.Suite
}

func (suite *TypeLogTimeFormatTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"   ",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeLogTimeFormatTestStruct{}))
		})
	}
}

func (suite *TypeLogTimeFormatTestSuite) TestUnmarshalOk() {
	testData := []string{
		"unix",
		"unix-ms",
		"unix-micro",
		"unix-nano",
		"rfc3339",
		"rfc3339-nano",
		"UNIX-MS",
		"RFC3339",
		"2006-01-02 15:04:05",
		"02 Jan 2006 15:04:05 MST",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.NoError(t, json.Unmarshal(data, &typeLogTimeFormatTestStruct{}))
		})
	}
}

func (suite *TypeLogTimeFormatTestSuite) TestZerologFormatPresets() {
	testData := []struct {
		value string
		want  string
	}{
		{value: "unix", want: zerolog.TimeFormatUnix},
		{value: "unix-ms", want: zerolog.TimeFormatUnixMs},
		{value: "unix-micro", want: zerolog.TimeFormatUnixMicro},
		{value: "unix-nano", want: zerolog.TimeFormatUnixNano},
		{value: "rfc3339", want: time.RFC3339},
		{value: "rfc3339-nano", want: time.RFC3339Nano},
		// Case-insensitive on presets.
		{value: "UNIX-NANO", want: zerolog.TimeFormatUnixNano},
		{value: "RFC3339-Nano", want: time.RFC3339Nano},
		// Surrounding whitespace is trimmed on presets.
		{value: "  unix-ms  ", want: zerolog.TimeFormatUnixMs},
		// Anything else is a Go layout passed through verbatim.
		{value: "2006-01-02 15:04:05", want: "2006-01-02 15:04:05"},
		{value: "02 Jan 2006 15:04:05 MST", want: "02 Jan 2006 15:04:05 MST"},
	}

	for _, c := range testData {
		c := c

		suite.T().Run(c.value, func(t *testing.T) {
			v := config.TypeLogTimeFormat{}
			assert.NoError(t, v.Set(c.value))
			assert.Equal(t, c.want, v.ZerologFormat())
		})
	}
}

func (suite *TypeLogTimeFormatTestSuite) TestSetRejectsEmpty() {
	v := config.TypeLogTimeFormat{}

	suite.Error(v.Set(""))
	suite.Error(v.Set("   "))
}

func (suite *TypeLogTimeFormatTestSuite) TestGetDefault() {
	// A zero value carries no format; Get must fall back to the default.
	v := config.TypeLogTimeFormat{}
	suite.Equal("unix-ms", v.Get("unix-ms"))

	suite.NoError(v.Set("rfc3339"))
	suite.Equal("rfc3339", v.Get("unix-ms"))
}

func (suite *TypeLogTimeFormatTestSuite) TestMarshalRoundTrip() {
	testStruct := &typeLogTimeFormatTestStruct{}
	suite.NoError(testStruct.Value.Set("rfc3339"))

	encoded, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.Contains(string(encoded), "rfc3339")
}

func TestTypeLogTimeFormat(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeLogTimeFormatTestSuite{})
}
