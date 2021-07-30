package config_test

import (
	"encoding/json"
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
	testData := []string{
		"",
		"1s",
		"1,",
		"1,2",
		".",
		"3.4.5",
		"3.5.",
		".3.5",
		"some word",
		"1e2",
		"-1.0",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeErrorRateTestStruct{}))
		})
	}
}

func (suite *TypeErrorRateTestSuite) TestUnmarshalOk() {
	data, err := json.Marshal(map[string]float64{
		"value": 1.0,
	})
	suite.NoError(err)

	testStruct := &typeErrorRateTestStruct{}
	suite.NoError(json.Unmarshal(data, testStruct))
	suite.InEpsilon(1.0, testStruct.Value.Value, 1e-10)
}

func (suite *TypeErrorRateTestSuite) TestMarshalOk() {
	testStruct := typeErrorRateTestStruct{
		Value: config.TypeErrorRate{
			Value: 1.01,
		},
	}

	encodedJSON, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": 1.01}`, string(encodedJSON))
}

func (suite *TypeErrorRateTestSuite) TestGet() {
	value := config.TypeErrorRate{}
	suite.InEpsilon(1.0, value.Get(1.0), 1e-10)

	value.Value = 5.0
	suite.InEpsilon(5.0, value.Get(1.0), 1e-10)
}

func TestTypeErrorRate(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeErrorRateTestSuite{})
}
