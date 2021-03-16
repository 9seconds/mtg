package config_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeStatsdTagFormatTestStruct struct {
	Value config.TypeStatsdTagFormat `json:"value"`
}

type TypeStatsdTagFormat struct {
	suite.Suite
}

func (suite *TypeStatsdTagFormat) TestUnmarshalFail() {
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
			assert.Error(t, json.Unmarshal(data, &typeStatsdTagFormatTestStruct{}))
		})
	}
}

func (suite *TypeStatsdTagFormat) TestUnmarshalOk() {
	testData := []string{
		config.TypeStatsdTagFormatDatadog,
		config.TypeStatsdTagFormatInfluxdb,
		config.TypeStatsdTagFormatGraphite,
		strings.ToUpper(config.TypeStatsdTagFormatDatadog),
		strings.ToUpper(config.TypeStatsdTagFormatInfluxdb),
		strings.ToUpper(config.TypeStatsdTagFormatGraphite),
		strings.ToLower(config.TypeStatsdTagFormatDatadog),
		strings.ToLower(config.TypeStatsdTagFormatInfluxdb),
		strings.ToLower(config.TypeStatsdTagFormatGraphite),
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeStatsdTagFormatTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.EqualValues(t,
				strings.ToLower(value),
				testStruct.Value.Value(config.TypeStatsdTagFormatDatadog))
		})
	}
}

func (suite *TypeStatsdTagFormat) TestMarshalOk() {
	testData := []string{
		config.TypeStatsdTagFormatDatadog,
		config.TypeStatsdTagFormatInfluxdb,
		config.TypeStatsdTagFormatGraphite,
		strings.ToUpper(config.TypeStatsdTagFormatDatadog),
		strings.ToUpper(config.TypeStatsdTagFormatInfluxdb),
		strings.ToUpper(config.TypeStatsdTagFormatGraphite),
		strings.ToLower(config.TypeStatsdTagFormatDatadog),
		strings.ToLower(config.TypeStatsdTagFormatInfluxdb),
		strings.ToLower(config.TypeStatsdTagFormatGraphite),
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeStatsdTagFormatTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, strings.ToLower(value), testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, strings.ToLower(value), string(marshalled))
		})
	}
}

func (suite *TypeStatsdTagFormat) TestValue() {
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

func TestTypeStatsdTagFormat(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeStatsdTagFormat{})
}
