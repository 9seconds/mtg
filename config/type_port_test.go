package config_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typePortTestStruct struct {
	Value config.TypePort `json:"value"`
}

type TypePortTestSuite struct {
	suite.Suite
}

func (suite *TypePortTestSuite) TestUnmarshalNil() {
	typ := &config.TypePort{}
	suite.NoError(typ.UnmarshalJSON(nil))
	suite.Equal("0", typ.String())
}

func (suite *TypePortTestSuite) TestUnmarshalFail() {
	testData := []int{
		-1,
		1_000_000,
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]int{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(strconv.Itoa(v), func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typePortTestStruct{}))
		})
	}
}

func (suite *TypePortTestSuite) TestUnmarshalOk() {
	testData := []int{
		1,
		1_000,
		65535,
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]int{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(strconv.Itoa(v), func(t *testing.T) {
			testStruct := &typePortTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.EqualValues(t, value, testStruct.Value.Value(0))
		})
	}
}

func (suite *TypePortTestSuite) TestMarshalOk() {
	testData := map[string]int{
		"1":     1,
		"1000":  1000,
		"65535": 65535,
	}

	for k, v := range testData {
		name := k
		value := v

		data, err := json.Marshal(map[string]int{
			"value": value,
		})
		suite.NoError(err)

		suite.T().Run(name, func(t *testing.T) {
			testStruct := &typePortTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, name, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalJSON()
			assert.NoError(t, err)
			assert.Equal(t, name, string(marshalled))
		})
	}
}

func (suite *TypePortTestSuite) TestValue() {
	testStruct := &typePortTestStruct{}

	suite.EqualValues(0, testStruct.Value.Value(0))
	suite.EqualValues(1, testStruct.Value.Value(1))

	data, err := json.Marshal(map[string]int{
		"value": 5,
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.EqualValues(5, testStruct.Value.Value(0))
	suite.EqualValues(5, testStruct.Value.Value(1))
}

func TestTypePort(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypePortTestSuite{})
}
