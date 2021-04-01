package config_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeURLTestStruct struct {
	Value config.TypeURL `json:"value"`
}

type TypeURLTestSuite struct {
	suite.Suite
}

func (suite *TypeURLTestSuite) TestUnmarshalNil() {
	u, _ := url.Parse("https://google.com")

	typ := &config.TypeURL{}
	suite.NoError(typ.UnmarshalText(nil))
	suite.Empty(typ.String())
	suite.Equal("https://google.com", typ.Value(u).String())
}

func (suite *TypeURLTestSuite) TestUnmarshalFail() {
	testData := []string{
		"http:/aaa.com",
		"ipv4",
		"111",
		"://111",
		"http://aaa.com:xxx",
		"gopher://aaa.com:888",
		"gopher://aaa.com",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeURLTestStruct{}))
		})
	}
}

func (suite *TypeURLTestSuite) TestUnmarshalOk() {
	testData := map[string]string{
		"https://10.0.0.10:80":    "https://10.0.0.10:80",
		"https://10.0.0.10:443":   "https://10.0.0.10",
		"http://10.0.0.10:8":      "http://10.0.0.10:8",
		"http://10.0.0.10:80":     "http://10.0.0.10",
		"socks5://10.0.0.10:1080": "socks5://10.0.0.10",
		"socks5://10.0.0.10:888":  "socks5://10.0.0.10:888",
	}

	for k, v := range testData {
		expected := k
		actual := v

		data, err := json.Marshal(map[string]string{
			"value": actual,
		})
		suite.NoError(err)

		suite.T().Run(actual, func(t *testing.T) {
			testStruct := &typeURLTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, expected, testStruct.Value.Value(nil).String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, expected, string(marshalled))
		})
	}
}

func (suite *TypeURLTestSuite) TestValue() {
	testStruct := &typeURLTestStruct{}

	u1, _ := url.Parse("https://10.0.0.10:80")
	u2, _ := url.Parse("https://10.1.0.10:80")

	suite.Equal("https://10.0.0.10:80", testStruct.Value.Value(u1).String())
	suite.Equal("https://10.1.0.10:80", testStruct.Value.Value(u2).String())

	data, err := json.Marshal(map[string]string{
		"value": "http://127.0.0.1:80",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.Equal("http://127.0.0.1:80", testStruct.Value.Value(u1).String())
	suite.Equal("http://127.0.0.1:80", testStruct.Value.Value(u2).String())
}

func TestTypeURL(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeURLTestSuite{})
}
