package config_test

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeHostPortTestStruct struct {
	Value config.TypeHostPort `json:"value"`
}

type TypeHostPortTestSuite struct {
	suite.Suite
}

func (suite *TypeHostPortTestSuite) TestUnmarshalFail() {
	testData := []string{
		"10.0.0.10:aaa",
		"10.0.0.10:",
		":",
		"xxx",
		"xxx:80",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeHostPortTestStruct{}))
		})
	}
}

func (suite *TypeHostPortTestSuite) TestUnmarshalOk() {
	testData := []string{
		"10.0.0.10:80",
		"0.0.0.0:80",
		":8000",
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeHostPortTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.EqualValues(t, value, testStruct.Value.Value(nil, 0))
		})
	}
}

func (suite *TypeHostPortTestSuite) TestMarshalOk() {
	testData := []string{
		"10.0.0.10:80",
		"0.0.0.0:80",
		":8000",
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeHostPortTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, value, string(marshalled))
		})
	}
}

func (suite *TypeHostPortTestSuite) TestValue() {
	testStruct := &typeHostPortTestStruct{}

	suite.EqualValues("127.0.0.1:80",
		testStruct.Value.Value(net.ParseIP("127.0.0.1"), 80))
	suite.EqualValues("127.1.0.1:80",
		testStruct.Value.Value(net.ParseIP("127.1.0.1"), 80))

	data, err := json.Marshal(map[string]string{
		"value": "127.0.0.1:80",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.EqualValues("127.0.0.1:80", testStruct.Value.Value(nil, 0))
	suite.EqualValues("127.0.0.1:80", testStruct.Value.Value(net.ParseIP("10.0.0.10"), 3000))
}

func TestTypeHostPort(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeHostPortTestSuite{})
}
