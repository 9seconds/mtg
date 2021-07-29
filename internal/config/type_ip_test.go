package config_test

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
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
		"",
		"....",
		"0...",
		"300.200.200.800",
		"[]",
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
	testData := map[string]string{
		"2001:0db8:85a3:0000:0000:8a2e:0370:7334": "2001:db8:85a3::8a2e:370:7334",
		"127.0.0.1": "127.0.0.1",
	}

	for k, v := range testData {
		expected := v

		data, err := json.Marshal(map[string]string{
			"value": k,
		})
		suite.NoError(err)

		suite.T().Run(k, func(t *testing.T) {
			testStruct := &typeIPTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, expected, testStruct.Value.Get(nil).String())
		})
	}
}

func (suite *TypeIPTestSuite) TestMarshalOk() {
	testData := []string{
		"2001:db8:85a3::8a2e:370:7334",
		"127.0.0.1",
	}

	for _, v := range testData {
		value := v

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeIPTestStruct{
				Value: config.TypeIP{
					Value: net.ParseIP(value),
				},
			}

			encodedJSON, err := json.Marshal(testStruct)
			assert.NoError(t, err)

			expectedJSON, err := json.Marshal(map[string]string{
				"value": value,
			})
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(encodedJSON))
		})
	}
}

func (suite *TypeIPTestSuite) TestGet() {
	value := config.TypeIP{}
	suite.Equal("127.0.0.1", value.Get(net.ParseIP("127.0.0.1")).String())

	suite.NoError(value.Set("127.0.0.2"))
	suite.Equal("127.0.0.2", value.Get(net.ParseIP("127.0.0.1")).String())
}

func TestTypeIP(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeIPTestSuite{})
}
