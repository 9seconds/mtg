package config_test

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeErrorRateTestStruct struct {
	Value config.TypeErrorRate `json:"value"`
}

type TypeErrorRateTestSuite struct {
	suite.Suite
}

func (suite *TypeErrorRateTestSuite) TestUnmarshalFail() {
	testData := []float64{
		1000,
		-100,
		-0.0001,
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]float64{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(strconv.FormatFloat(v, 'f', -1, 64), func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeErrorRateTestStruct{}))
		})
	}

	data, err := json.Marshal(map[string]string{
		"value": "hello",
	})
	suite.NoError(err)
	suite.Error(json.Unmarshal(data, &typeErrorRateTestStruct{}))
}

func (suite *TypeErrorRateTestSuite) TestUnmarshalOk() {
	testData := []float64{
		1,
		55.5,
		0.0001,
		1e-6,
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]float64{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(strconv.FormatFloat(v, 'f', -1, 64), func(t *testing.T) {
			testStruct := &typeErrorRateTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.InEpsilon(t, value, testStruct.Value.Value(0), 1e-10)
		})
	}
}

func (suite *TypeErrorRateTestSuite) TestMarshalOk() {
	testData := []float64{
		1,
		55.5,
		0.0001,
		1e-6,
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]float64{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(strconv.FormatFloat(v, 'f', -1, 64), func(t *testing.T) {
			testStruct := &typeErrorRateTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))

			parsed, err := strconv.ParseFloat(testStruct.Value.String(), 64)
			assert.NoError(t, err)
			assert.InEpsilon(t, value, parsed, 1e-10)

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)

			parsed, err = strconv.ParseFloat(string(marshalled), 64)
			assert.NoError(t, err)
			assert.InEpsilon(t, value, parsed, 1e-10)
		})
	}
}

func (suite *TypeErrorRateTestSuite) TestValue() {
	testStruct := &typeErrorRateTestStruct{}

	suite.InEpsilon(1, testStruct.Value.Value(1), 1e-10)
	suite.InEpsilon(2, testStruct.Value.Value(2), 1e-10)

	data, err := json.Marshal(map[string]float64{
		"value": 1,
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.InEpsilon(1, testStruct.Value.Value(2), 1e-10)
	suite.InEpsilon(1, testStruct.Value.Value(3), 1e-10)
}

func TestTypeErrorRate(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeErrorRateTestSuite{})
}
