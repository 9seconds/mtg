package config_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/config"
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
		"1t",
		"1",
		"-1s",
		"-1h",
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
		"1m":   time.Minute,
		"2h1s": 2*time.Hour + time.Second,
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
			assert.Equal(t, value, testStruct.Value.Value(0))
		})
	}
}

func (suite *TypeDurationTestSuite) TestMarshalOk() {
	testData := []string{
		"1s",
		"1m0s",
		"2h0m1s",
	}

	for _, v := range testData {
		name := v

		data, err := json.Marshal(map[string]string{
			"value": name,
		})
		suite.NoError(err)

		suite.T().Run(name, func(t *testing.T) {
			testStruct := &typeDurationTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, name, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, string(marshalled), name)
		})
	}
}

func (suite *TypeDurationTestSuite) TestValue() {
	testStruct := &typeDurationTestStruct{}

	suite.EqualValues(0, testStruct.Value.Value(0))
	suite.Equal(time.Second, testStruct.Value.Value(time.Second))

	data, err := json.Marshal(map[string]string{
		"value": "1s",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.Equal(time.Second, testStruct.Value.Value(0))
	suite.Equal(time.Second, testStruct.Value.Value(time.Minute))
}

func TestTypeDuration(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeDurationTestSuite{})
}
