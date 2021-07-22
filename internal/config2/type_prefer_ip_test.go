package config2_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typePreferIPTestStruct struct {
	Value config2.TypePreferIP `json:"value"`
}

type TypePreferIPTestSuite struct {
	suite.Suite
}

func (suite *TypePreferIPTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"prefer",
		"preferipv4",
		config2.TypePreferIPPreferIPv4 + "_",
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
		config2.TypePreferIPPreferIPv4,
		config2.TypePreferIPPreferIPv6,
		config2.TypePreferOnlyIPv4,
		config2.TypePreferOnlyIPv6,
		strings.ToTitle(config2.TypePreferOnlyIPv4),
		strings.ToTitle(config2.TypePreferOnlyIPv6),
		strings.ToTitle(config2.TypePreferIPPreferIPv4),
		strings.ToTitle(config2.TypePreferIPPreferIPv6),
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
		config2.TypePreferIPPreferIPv4,
		config2.TypePreferIPPreferIPv6,
		config2.TypePreferOnlyIPv4,
		config2.TypePreferOnlyIPv6,
	}

	for _, v := range testData {
		value := v

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typePreferIPTestStruct{
				Value: config2.TypePreferIP{
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
	value := config2.TypePreferIP{}
	suite.Equal(config2.TypePreferIPPreferIPv4,
		value.Get(config2.TypePreferIPPreferIPv4))

	suite.NoError(value.Set(config2.TypePreferIPPreferIPv6))
	suite.Equal(config2.TypePreferIPPreferIPv6,
		value.Get(config2.TypePreferIPPreferIPv4))
}

func TestTypePreferIP(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypePreferIPTestSuite{})
}
