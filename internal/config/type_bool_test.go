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
	testData := []string{
		"",
		"np",
		"нет",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeBoolTestStruct{}))
		})
	}
}

func (suite *TypeBoolTestSuite) TestUnmarshalOk() {
	testData := map[string]bool{
		"0":        false,
		"N":        false,
		"nO":       false,
		"no":       false,
		"dISAbLEd": false,
		"False":    false,
		"false":    false,

		"1":       true,
		"y":       true,
		"Yes":     true,
		"yes":     true,
		"enABLED": true,
		"True":    true,
		"TRUE":    true,
		"true":    true,
	}

	for k, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": k,
		})
		suite.NoError(err)

		suite.T().Run(k, func(t *testing.T) {
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
		name := strconv.FormatBool(v)

		suite.T().Run(name, func(t *testing.T) {
			testStruct := typeBoolTestStruct{
				Value: config.TypeBool{
					Value: v,
				},
			}

			data, err := json.Marshal(testStruct)
			assert.NoError(t, err)
			assert.JSONEq(t, fmt.Sprintf(`{"value": "%s"}`, name), string(data))
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
