package config2_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeHTTPPathTestStruct struct {
	Value config2.TypeHTTPPath `json:"value"`
}

type TypeHTTPPathTestSuite struct {
	suite.Suite
}

func (suite *TypeHTTPPathTestSuite) TestUnmarshalOk() {
	testData := map[string]string{
		"":      "/",
		"/":     "/",
		"/path": "/path",
		"path":  "/path",
	}

	for k, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": k,
		})
		suite.NoError(err)

		suite.T().Run(k, func(t *testing.T) {
			testStruct := &typeHTTPPathTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.Get(""))
		})
	}
}

func (suite *TypeHTTPPathTestSuite) TestMarshalOk() {
	value := typeHTTPPathTestStruct{
		Value: config2.TypeHTTPPath{
			Value: "/path",
		},
	}

	data, err := json.Marshal(value)
	suite.NoError(err)
	suite.JSONEq(`{"value": "/path"}`, string(data))
}

func (suite *TypeHTTPPathTestSuite) TestGet() {
	value := config2.TypeHTTPPath{}
	suite.Equal("/hello", value.Get("/hello"))

	suite.NoError(value.Set("/lalala"))
	suite.Equal("/lalala", value.Get("/hello"))
}

func TestTypeHTTPPath(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeHTTPPathTestSuite{})
}
