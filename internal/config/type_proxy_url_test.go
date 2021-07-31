package config_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeProxyURLTestStruct struct {
	Value config.TypeProxyURL `json:"value"`
}

type ProxyURLTestSuite struct {
	suite.Suite
}

func (suite *ProxyURLTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"socks5://",
		"://lala",
		"/path",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeProxyURLTestStruct{}))
		})
	}
}

func (suite *ProxyURLTestSuite) TestUnmarshalOk() {
	testData := map[string]string{
		"socks5://127.0.0.1/?open_threshold=1": "socks5://127.0.0.1:1080/?open_threshold=1",
		"socks5://127.0.0.1:80":                "socks5://127.0.0.1:80",
	}

	for k, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": k,
		})
		suite.NoError(err)

		suite.T().Run(k, func(t *testing.T) {
			testStruct := &typeProxyURLTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))

			parsed, _ := url.Parse(value)

			assert.Equal(t, parsed.Scheme, testStruct.Value.Get(nil).Scheme)
			assert.Equal(t, parsed.Host, testStruct.Value.Get(nil).Host)
			assert.Equal(t, parsed.RawQuery, testStruct.Value.Get(nil).RawQuery)
			assert.Equal(t, parsed.Path, testStruct.Value.Get(nil).Path)
		})
	}
}

func (suite *ProxyURLTestSuite) TestMarshalOk() {
	parsed, _ := url.Parse("socks5://127.0.0.1:1080?open_threshold=1")
	testStruct := &typeProxyURLTestStruct{
		Value: config.TypeProxyURL{
			Value: parsed,
		},
	}

	encodedJSON, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": "socks5://127.0.0.1:1080?open_threshold=1"}`,
		string(encodedJSON))
}

func (suite *ProxyURLTestSuite) TestGet() {
	emptyURL := &url.URL{}

	value := config.TypeProxyURL{}
	suite.Equal(emptyURL, value.Get(emptyURL))

	value.Value = &url.URL{}
	suite.Equal(value.Value, value.Get(emptyURL))
}

func TestTypeProxyURL(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProxyURLTestSuite{})
}
