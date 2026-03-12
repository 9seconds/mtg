package config_test

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeHttpsURLTestStruct struct {
	Value config.TypeHttpsURL `json:"value"`
}

type HttpsURLTestSuite struct {
	suite.Suite
}

func (suite *HttpsURLTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"https://",
		"://lala",
		"/path",
		"http://example.com",
		"socks5://example.com",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeHttpsURLTestStruct{}))
		})
	}
}

func (suite *HttpsURLTestSuite) TestUnmarshalOk() {
	testData := map[string]string{
		"https://example.com":            "https://example.com",
		"https://example.com:8443":       "https://example.com:8443",
		"https://example.com/path?q=1":   "https://example.com/path?q=1",
		"https://user:pass@example.com":  "https://user:pass@example.com",
	}

	for k, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": k,
		})
		suite.NoError(err)

		suite.T().Run(k, func(t *testing.T) {
			testStruct := &typeHttpsURLTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))

			parsed, _ := url.Parse(value)

			assert.Equal(t, parsed.Scheme, testStruct.Value.Get(nil).Scheme)
			assert.Equal(t, parsed.Host, testStruct.Value.Get(nil).Host)
			assert.Equal(t, parsed.RawQuery, testStruct.Value.Get(nil).RawQuery)
			assert.Equal(t, parsed.Path, testStruct.Value.Get(nil).Path)
		})
	}
}

func (suite *HttpsURLTestSuite) TestMarshalOk() {
	parsed, _ := url.Parse("https://example.com/path?q=1")
	testStruct := &typeHttpsURLTestStruct{
		Value: config.TypeHttpsURL{
			Value: parsed,
		},
	}

	encodedJSON, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": "https://example.com/path?q=1"}`,
		string(encodedJSON))
}

func (suite *HttpsURLTestSuite) TestGet() {
	emptyURL := &url.URL{}

	value := config.TypeHttpsURL{}
	suite.Equal(emptyURL, value.Get(emptyURL))

	value.Value = &url.URL{}
	suite.Equal(value.Value, value.Get(emptyURL))
}

func TestTypeHttpsURL(t *testing.T) {
	t.Parallel()
	suite.Run(t, &HttpsURLTestSuite{})
}
