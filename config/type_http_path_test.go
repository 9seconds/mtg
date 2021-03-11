package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeHTTPPathTestStruct struct {
	Value config.TypeHTTPPath `json:"value"`
}

type TypeHTTPPathTestSuite struct {
	suite.Suite
}

func (suite *TypeHTTPPathTestSuite) TestUnmarshal() {
	testData := []string{
		"/hello",
		"hello",
		"hello/",
		"/hello/",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeHTTPPathTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, "/hello", testStruct.Value.Value(""))
		})
	}
}

func (suite *TypeHTTPPathTestSuite) TestMarshalOk() {
	testData := map[string]string{
		"":        "/",
		"/hello":  "/hello",
		"/hello/": "/hello",
		"hello/":  "/hello",
		"hello":   "/hello",
	}

	for k, v := range testData {
		toPass := k
		compareWith := v

		data, err := json.Marshal(map[string]string{
			"value": toPass,
		})
		suite.NoError(err)

		suite.T().Run(toPass, func(t *testing.T) {
			testStruct := &typeHTTPPathTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, compareWith, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, compareWith, string(marshalled))
		})
	}
}

func (suite *TypeHTTPPathTestSuite) TestValue() {
	testStruct := &typeHTTPPathTestStruct{}

	suite.Equal("/hello", testStruct.Value.Value("/hello"))

	data, err := json.Marshal(map[string]string{
		"value": "/map",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.Equal("/map", testStruct.Value.Value("/hello"))
}

func TestTypeHTTPPath(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeHTTPPathTestSuite{})
}
