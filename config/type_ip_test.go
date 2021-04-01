package config_test

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeIPTestStruct struct {
	Value config.TypeIP `json:"value"`
}

type TypeIPTestSuite struct {
	suite.Suite
}

func (suite *TypeIPTestSuite) TestUnmarshalFail() {
	testData := []string{
		"0.0.10",
		"10.0.0.10:",
		"xxx:80",
		"2001:0db8:85a3:0000:0000:8a2e:4",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeIPTestStruct{}))
		})
	}
}

func (suite *TypeIPTestSuite) TestUnmarshalOk() {
	testData := []string{
		"0.0.0.0",
		"10.0.0.10",
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeIPTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t,
				net.ParseIP(value).String(),
				testStruct.Value.Value(nil).String())
		})
	}
}

func (suite *TypeIPTestSuite) TestMarshalOk() {
	testData := []string{
		"0.0.0.0",
		"10.0.0.10",
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334",
	}

	for _, v := range testData {
		value := net.ParseIP(v).String()

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeIPTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, value, string(marshalled))
		})
	}
}

func (suite *TypeIPTestSuite) TestValue() {
	testStruct := &typeIPTestStruct{}
	suite.Empty(testStruct.Value.String())

	suite.Nil(testStruct.Value.Value(nil))
	suite.Equal("127.1.0.1", testStruct.Value.Value(net.ParseIP("127.1.0.1")).String())

	data, err := json.Marshal(map[string]string{
		"value": "127.0.0.1",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.Equal("127.0.0.1", testStruct.Value.Value(nil).String())
	suite.Equal("127.0.0.1", testStruct.Value.Value(net.ParseIP("10.0.0.10")).String())
}

func TestTypeIP(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeIPTestSuite{})
}
