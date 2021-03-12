package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeMetricPrefixTestStruct struct {
	Value config.TypeMetricPrefix `json:"value"`
}

type TypeMetricPrefixTestSuite struct {
	suite.Suite
}

func (suite *TypeMetricPrefixTestSuite) TestUnmarshalFail() {
	testData := []string{
		"aaa.aaa",
		"aaa-bbb",
		"aaa:ccc",
		"metric prefix",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeMetricPrefixTestStruct{}))
		})
	}
}

func (suite *TypeMetricPrefixTestSuite) TestUnmarshalOk() {
	testData := []string{
		"mtg",
		"mtg111",
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeMetricPrefixTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.Value(""))
		})
	}
}

func (suite *TypeMetricPrefixTestSuite) TestMarshalOk() {
	testData := []string{
		"mtg",
		"mtg111",
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeMetricPrefixTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, value, string(marshalled))
		})
	}
}

func (suite *TypeMetricPrefixTestSuite) TestValue() {
	testStruct := &typeMetricPrefixTestStruct{}

	suite.Equal("mtg", testStruct.Value.Value("mtg"))
	suite.Equal("vvv", testStruct.Value.Value("vvv"))

	data, err := json.Marshal(map[string]string{
		"value": "aaa",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.Equal("aaa", testStruct.Value.Value("mtg"))
	suite.Equal("aaa", testStruct.Value.Value("vvv"))
}

func TestTypeMetricPrefix(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeMetricPrefixTestSuite{})
}
