package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeDCTestStruct struct {
	Value config.TypeDC `json:"value"`
}

type TypeDCTestSuite struct {
	suite.Suite
}

func (suite *TypeDCTestSuite) TestUnmarshalFail() {
	testData := []string{
		"-1s",
		"1202002020202",
		"xxx",
		"-11111111111111",
		"",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeDCTestStruct{}))
		})
	}
}

func (suite *TypeDCTestSuite) TestUnmarshalOk() {
	testData := map[string]int{
		"1":   1,
		"-1":  1,
		"203": 203,
	}

	for value, expected := range testData {
		data, err := json.Marshal(map[string]string{
			"value": value,
		})
		suite.NoError(err)

		suite.T().Run(value, func(t *testing.T) {
			testStruct := &typeDCTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, expected, testStruct.Value.Value)
			assert.Equal(t, expected, testStruct.Value.Get())
		})
	}
}

func (suite *TypeDCTestSuite) TestMarshalOk() {
	testData := map[string]string{
		"1":   "1",
		"203": "203",
	}

	for k, v := range testData {
		value := k
		expected := v

		suite.T().Run(value, func(t *testing.T) {
			testStruct := &typeDCTestStruct{}

			assert.NoError(t, testStruct.Value.Set(value))

			data, err := json.Marshal(testStruct)
			assert.NoError(t, err)

			expectedJSON, err := json.Marshal(map[string]string{
				"value": expected,
			})
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(data))
		})
	}
}

func TestTypeDC(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeDCTestSuite{})
}
