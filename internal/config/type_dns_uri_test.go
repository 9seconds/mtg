package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeDNSURITestStruct struct {
	Value config.TypeDNSURI `json:"value"`
}

type TypeDNSURITestSuite struct {
	suite.Suite
}

func (suite *TypeDNSURITestSuite) TestUnmarshalFail() {
	testData := []string{
		"xx",
		"ppar",
		"",
		"dns://hahaha",
		"udp://xcxxcv",
		"udp://1.1.1.1/xcv",
		"1.1.1.1/xxx",
		"tls://dns/xx",
		"tls://1.1.1.1/xx",
		"https://user:password@1.1.1.1",
		"tls://user:password@1.1.1.1",
		"udp://user:password@1.1.1.1",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeDNSURITestStruct{}))
		})
	}
}

func (suite *TypeDNSURITestSuite) TestUnmarshalOk() {
	testData := []string{
		"1.1.1.1",
		"tls://1.1.1.1",
		"tls://dns.google",
		"https://1.1.1.1",
		"https://1.1.1.1/dns-query",
		"https://dns.google",
		"https://dns.google/dns-query",
		"udp://1.1.1.1",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeDNSURITestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))
			if v == "1.1.1.1" {
				v = "udp://" + v
			}
			assert.Equal(t, v, testStruct.Value.String())
		})
	}
}

func (suite *TypeDNSURITestSuite) TestMarshalOk() {
	testData := []string{
		"tls://1.1.1.1",
		"tls://dns.google",
		"https://1.1.1.1",
		"https://1.1.1.1/dns-query",
	}

	for _, v := range testData {
		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typePreferIPTestStruct{
				Value: config.TypePreferIP{
					Value: v,
				},
			}

			encodedJSON, err := json.Marshal(testStruct)
			assert.NoError(t, err)

			expectedJSON, err := json.Marshal(map[string]string{
				"value": v,
			})
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(encodedJSON))
		})
	}
}

func (suite *TypeDNSURITestSuite) TestGet() {
	value := config.TypeDNSURI{}
	suite.Nil(value.Get(nil))

	suite.NoError(value.Set("tls://1.1.1.1"))
	suite.NotNil(value.Get(nil))
}

func TestDNSURI(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeDNSURITestSuite{})
}
