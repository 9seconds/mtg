package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type TypeURLHostTestStruct struct {
	Value config.TypeURLHost `json:"value"`
}

type TypeURLHostTestSuite struct {
	suite.Suite
}

func (suite *TypeURLHostTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &TypeURLHostTestStruct{}))
		})
	}
}

func (suite *TypeURLHostTestSuite) TestUnmarshalOk() {
	testData := []string{
		"dns.google",
		"cloudflare-dns.com",
		"9.9.9.9",
		"127.0.0.1:80",
		"10.0.0.10:6553",
		"[2001:1234::1]:6553",
		"[::1]:80",
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &TypeURLHostTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, value, testStruct.Value.Value)
		})
	}
}

func (suite *TypeURLHostTestSuite) TestMarshalOk() {
	testStruct := TypeURLHostTestStruct{
		Value: config.TypeURLHost{
			Value: "127.0.0.1:8000",
		},
	}

	data, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": "127.0.0.1:8000"}`, string(data))
}

func (suite *TypeURLHostTestSuite) TestGet() {
	value := config.TypeURLHost{}
	suite.Equal("127.0.0.1:9000", value.Get("127.0.0.1:9000"))

	value.Value = "127.0.0.1:80"
	suite.Equal("127.0.0.1:80", value.Get("127.0.0.1:9000"))
}

func TestTypeURLHost(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeURLHostTestSuite{})
}
