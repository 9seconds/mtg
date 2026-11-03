package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeHostTestStruct struct {
	Value config.TypeHost `json:"value"`
}

type TypeHostTestSuite struct {
	suite.Suite
}

func (suite *TypeHostTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"web:8443",
		"http://example.com",
		"example.com/path",
		"two words",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeHostTestStruct{}))
		})
	}
}

func (suite *TypeHostTestSuite) TestUnmarshalOk() {
	testData := []string{
		"example.com",
		"web",
		"sub.example.com",
		"127.0.0.1",
		"2001:db8::1",
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": value,
		})
		suite.NoError(err)

		suite.T().Run(value, func(t *testing.T) {
			testStruct := &typeHostTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.Get(""))
		})
	}
}

func (suite *TypeHostTestSuite) TestGet() {
	value := config.TypeHost{}
	suite.Equal("default", value.Get("default"))

	suite.NoError(value.Set("example.com"))
	suite.Equal("example.com", value.Get("default"))
}

func TestTypeHost(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeHostTestSuite{})
}
