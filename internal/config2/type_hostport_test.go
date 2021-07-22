package config2_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeHostPortTestStruct struct {
	Value config2.TypeHostPort `json:"value"`
}

type TypeHostPortTestSuite struct {
	suite.Suite
}

func (suite *TypeHostPortTestSuite) TestUnmarshalFail() {
	testData := []string{
		":",
		":800",
		"127.0.0.1:8000000",
		"12...:80",
		"",
		"localhost",
		"google.com:",
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
		"127.0.0.1:80",
		"10.0.0.10:6553",
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
			assert.Equal(t, value, testStruct.Value.Value)
		})
	}
}

func (suite *TypeHostPortTestSuite) TestMarshalOk() {
	testStruct := typeHostPortTestStruct{
		Value: config2.TypeHostPort{
			Value: "127.0.0.1:8000",
		},
	}

	data, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": "127.0.0.1:8000"}`, string(data))
}

func (suite *TypeHostPortTestSuite) TestGet() {
	value := config2.TypeHostPort{}
	suite.Equal("127.0.0.1:9000", value.Get("127.0.0.1:9000"))

	value.Value = "127.0.0.1:80"
	suite.Equal("127.0.0.1:80", value.Get("127.0.0.1:9000"))
}

func TestTypeHostPort(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeHostPortTestSuite{})
}
