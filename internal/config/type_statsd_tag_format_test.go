package config_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeStatsdTagFormatTestStruct struct {
	Value config.TypeStatsdTagFormat `json:"value"`
}

type StatsdTagFormatTestSuite struct {
	suite.Suite
}

func (suite *StatsdTagFormatTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"dogdog",
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

func (suite *StatsdTagFormatTestSuite) TestUnmarshalOk() {
	testData := []string{
		config.TypeStatsdTagFormatInfluxdb,
		config.TypeStatsdTagFormatGraphite,
		config.TypeStatsdTagFormatDatadog,
		strings.ToUpper(config.TypeStatsdTagFormatInfluxdb),
		strings.ToUpper(config.TypeStatsdTagFormatGraphite),
		strings.ToUpper(config.TypeStatsdTagFormatDatadog),
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
			assert.Equal(t, strings.ToLower(value), testStruct.Value.Value)
		})
	}
}

func (suite *StatsdTagFormatTestSuite) TestMarshalOk() {
	testData := []string{
		config.TypeStatsdTagFormatInfluxdb,
		config.TypeStatsdTagFormatGraphite,
		config.TypeStatsdTagFormatDatadog,
	}

	for _, v := range testData {
		value := v

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeStatsdTagFormatTestStruct{
				Value: config.TypeStatsdTagFormat{
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

func (suite *StatsdTagFormatTestSuite) TestGet() {
	value := config.TypeStatsdTagFormat{}
	suite.Equal(config.TypeStatsdTagFormatDatadog,
		value.Get(config.TypeStatsdTagFormatDatadog))

	suite.NoError(value.Set(config.TypeStatsdTagFormatInfluxdb))
	suite.Equal(config.TypeStatsdTagFormatInfluxdb,
		value.Get(config.TypeStatsdTagFormatDatadog))
}

func TestTypeStatsdTagFormat(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StatsdTagFormatTestSuite{})
}
