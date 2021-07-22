package config2_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeMetricPrefixTestStruct struct {
	Value config2.TypeMetricPrefix `json:"value"`
}

type TypeMetricPrefixTestSuite struct {
	suite.Suite
}

func (suite *TypeMetricPrefixTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"-",
		"hello/world",
		"lala*",
		"++sdf++",
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
	testStruct := &typeMetricPrefixTestStruct{}
	suite.NoError(json.Unmarshal([]byte(`{"value": "mtg"}`), testStruct))
	suite.Equal("mtg", testStruct.Value.Get("lalala"))
}

func (suite *TypeMetricPrefixTestSuite) TestMarshalOk() {
	testStruct := &typeMetricPrefixTestStruct{
		Value: config2.TypeMetricPrefix{
			Value: "mtg",
		},
	}

	data, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": "mtg"}`, string(data))
}

func (suite *TypeMetricPrefixTestSuite) TestGet() {
	value := config2.TypeMetricPrefix{}
	suite.Equal("lalala", value.Get("lalala"))

	value.Value = "mtg"
	suite.Equal("mtg", value.Get("lalala"))
}

func TestTypeMetricPrefix(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeMetricPrefixTestSuite{})
}
