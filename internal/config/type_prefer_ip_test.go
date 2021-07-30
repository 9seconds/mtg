package config_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typePreferIPTestStruct struct {
	Value config.TypePreferIP `json:"value"`
}

type TypePreferIPTestSuite struct {
	suite.Suite
}

func (suite *TypePreferIPTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"prefer",
		"preferipv4",
		config.TypePreferIPPreferIPv4 + "_",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typePreferIPTestStruct{}))
		})
	}
}

func (suite *TypePreferIPTestSuite) TestUnmarshalOk() {
	testData := []string{
		config.TypePreferIPPreferIPv4,
		config.TypePreferIPPreferIPv6,
		config.TypePreferOnlyIPv4,
		config.TypePreferOnlyIPv6,
		strings.ToTitle(config.TypePreferOnlyIPv4),
		strings.ToTitle(config.TypePreferOnlyIPv6),
		strings.ToTitle(config.TypePreferIPPreferIPv4),
		strings.ToTitle(config.TypePreferIPPreferIPv6),
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typePreferIPTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, strings.ToLower(value), testStruct.Value.Value)
		})
	}
}

func (suite *TypePreferIPTestSuite) TestMarshalOk() {
	testData := []string{
		config.TypePreferIPPreferIPv4,
		config.TypePreferIPPreferIPv6,
		config.TypePreferOnlyIPv4,
		config.TypePreferOnlyIPv6,
	}

	for _, v := range testData {
		value := v

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typePreferIPTestStruct{
				Value: config.TypePreferIP{
					Value: value,
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

func (suite *TypePreferIPTestSuite) TestGet() {
	value := config.TypePreferIP{}
	suite.Equal(config.TypePreferIPPreferIPv4,
		value.Get(config.TypePreferIPPreferIPv4))

	suite.NoError(value.Set(config.TypePreferIPPreferIPv6))
	suite.Equal(config.TypePreferIPPreferIPv6,
		value.Get(config.TypePreferIPPreferIPv4))
}

func TestTypePreferIP(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypePreferIPTestSuite{})
}
