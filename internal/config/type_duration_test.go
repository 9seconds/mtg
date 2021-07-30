package config_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeDurationTestStruct struct {
	Value config.TypeDuration `json:"value"`
}

type TypeDurationTestSuite struct {
	suite.Suite
}

func (suite *TypeDurationTestSuite) TestUnmarshalFail() {
	testData := []string{
		"-1s",
		"1 seconds ago",
		"1s ago",
		"",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeDurationTestStruct{}))
		})
	}
}

func (suite *TypeDurationTestSuite) TestUnmarshalOk() {
	testData := map[string]time.Duration{
		"1s":   time.Second,
		"0":    0 * time.Second,
		"0s":   0 * time.Second,
		"1\tM": time.Minute,
		"1H":   time.Hour,
		"1 h":  time.Hour,
	}

	for k, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": k,
		})
		suite.NoError(err)

		suite.T().Run(k, func(t *testing.T) {
			testStruct := &typeDurationTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.Value)
		})
	}
}

func (suite *TypeDurationTestSuite) TestMarshalOk() {
	testData := map[string]string{
		"1s":  "1s",
		"0":   "",
		"0s":  "",
		"0ms": "",
		"1 H": "1h0m0s",
	}

	for k, v := range testData {
		value := k
		expected := v

		suite.T().Run(value, func(t *testing.T) {
			testStruct := &typeDurationTestStruct{}

			assert.NoError(t, testStruct.Value.Set(value))

			data, err := json.Marshal(testStruct)
			assert.NoError(t, err)

			expectedJSON, err := json.Marshal(map[string]string{
				"value": expected,
			})
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(data))
		})
	}
}

func (suite *TypeDurationTestSuite) TestGet() {
	value := config.TypeDuration{}
	suite.Equal(time.Second, value.Get(time.Second))

	value.Value = 3 * time.Second
	suite.Equal(3*time.Second, value.Get(time.Hour))
}

func TestTypeDuration(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeDurationTestSuite{})
}
