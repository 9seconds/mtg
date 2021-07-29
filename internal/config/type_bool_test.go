package config_test

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeBoolTestStruct struct {
	Value config.TypeBool `json:"value"`
}

type TypeBoolTestSuite struct {
	suite.Suite
}

func (suite *TypeBoolTestSuite) TestUnmarshalFail() {
	testData := []interface{}{
		"",
		"np",
		"нет",
		int(10),
		[]int{},
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]interface{}{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(fmt.Sprintf("%v", v), func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeBoolTestStruct{}))
		})
	}
}

func (suite *TypeBoolTestSuite) TestUnmarshalOk() {
	testData := []bool{
		true,
		false,
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]bool{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(strconv.FormatBool(v), func(t *testing.T) {
			testStruct := &typeBoolTestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))

			if value {
				assert.True(t, testStruct.Value.Value)
			} else {
				assert.False(t, testStruct.Value.Value)
			}
		})
	}
}

func (suite *TypeBoolTestSuite) TestMarshalOk() {
	for _, v := range []bool{true, false} {
		value := v

		suite.T().Run(strconv.FormatBool(v), func(t *testing.T) {
			testStruct := typeBoolTestStruct{
				Value: config.TypeBool{
					Value: value,
				},
			}

			encodedJSON, err := json.Marshal(testStruct)
			assert.NoError(t, err)

			expectedJSON, err := json.Marshal(map[string]bool{
				"value": value,
			})
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(encodedJSON))
		})
	}
}

func (suite *TypeBoolTestSuite) TestGet() {
	value := config.TypeBool{}
	suite.False(value.Get(false))
	suite.True(value.Get(true))

	value.Value = true
	suite.True(value.Get(false))
	suite.True(value.Get(true))
}

func TestTypeBool(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeBoolTestSuite{})
}
