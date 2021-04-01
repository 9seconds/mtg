package config_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typePreferIPTestStruct struct {
	Value config.TypePreferIP `json:"value"`
}

type TypePreferIPTestSuite struct {
	suite.Suite
}

func (suite *TypePreferIPTestSuite) TestUnmarshalNil() {
	typ := &config.TypePreferIP{}
	suite.NoError(typ.UnmarshalText(nil))
	suite.Empty(typ.String())
}

func (suite *TypePreferIPTestSuite) TestUnmarshalFail() {
	testData := []string{
		"p",
		"ipv4",
		"onlyipv4",
		"ipv6prefer",
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
		strings.ToUpper(config.TypePreferIPPreferIPv4),
		strings.ToUpper(config.TypePreferIPPreferIPv6),
		strings.ToUpper(config.TypePreferOnlyIPv4),
		strings.ToUpper(config.TypePreferOnlyIPv6),
		strings.ToLower(config.TypePreferIPPreferIPv4),
		strings.ToLower(config.TypePreferIPPreferIPv6),
		strings.ToLower(config.TypePreferOnlyIPv4),
		strings.ToLower(config.TypePreferOnlyIPv6),
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
			assert.EqualValues(t,
				strings.ToLower(value),
				testStruct.Value.Value(config.TypePreferIPPreferIPv4))
		})
	}
}

func (suite *TypePreferIPTestSuite) TestMarshalOk() {
	testData := []string{
		config.TypePreferIPPreferIPv4,
		config.TypePreferIPPreferIPv6,
		config.TypePreferOnlyIPv4,
		config.TypePreferOnlyIPv6,
		strings.ToUpper(config.TypePreferIPPreferIPv4),
		strings.ToUpper(config.TypePreferIPPreferIPv6),
		strings.ToUpper(config.TypePreferOnlyIPv4),
		strings.ToUpper(config.TypePreferOnlyIPv6),
		strings.ToLower(config.TypePreferIPPreferIPv4),
		strings.ToLower(config.TypePreferIPPreferIPv6),
		strings.ToLower(config.TypePreferOnlyIPv4),
		strings.ToLower(config.TypePreferOnlyIPv6),
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
			assert.Equal(t, strings.ToLower(value), testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, strings.ToLower(value), string(marshalled))
		})
	}
}

func (suite *TypePreferIPTestSuite) TestValue() {
	testStruct := &typePreferIPTestStruct{}

	suite.EqualValues(config.TypePreferIPPreferIPv4,
		testStruct.Value.Value(config.TypePreferIPPreferIPv4))
	suite.EqualValues(config.TypePreferIPPreferIPv6,
		testStruct.Value.Value(config.TypePreferIPPreferIPv6))

	data, err := json.Marshal(map[string]string{
		"value": config.TypePreferOnlyIPv4,
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.EqualValues(config.TypePreferOnlyIPv4,
		testStruct.Value.Value(config.TypePreferOnlyIPv6))
	suite.EqualValues(config.TypePreferOnlyIPv4,
		testStruct.Value.Value(config.TypePreferIPPreferIPv6))
}

func TestTypePreferIP(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypePreferIPTestSuite{})
}
