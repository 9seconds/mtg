package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeConcurrencyTestStruct struct {
	Value config.TypeConcurrency `json:"value"`
}

type TypeConcurrencyTestSuite struct {
	suite.Suite
}

func (suite *TypeConcurrencyTestSuite) TestUnmarshalFail() {
	testData := []string{
		"-1",
		"0",
		"0.0",
		"1.0",
		"1.1",
		".",
		"some_value",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeConcurrencyTestStruct{}))
		})
	}
}

func (suite *TypeConcurrencyTestSuite) TestUnmarshalOk() {
	testStruct := &typeConcurrencyTestStruct{}

	suite.NoError(json.Unmarshal([]byte(`{"value": 1}`), testStruct))
	suite.EqualValues(1, testStruct.Value.Get(2))
}

func (suite *TypeConcurrencyTestSuite) TestMarshalOk() {
	testStruct := &typeConcurrencyTestStruct{
		Value: config.TypeConcurrency{
			Value: 2,
		},
	}

	data, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": 2}`, string(data))
}

func (suite *TypeConcurrencyTestSuite) TestGet() {
	value := config.TypeConcurrency{}
	suite.EqualValues(1, value.Get(1))

	value.Value = 3
	suite.EqualValues(3, value.Get(1))
}

func TestTypeConcurrency(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeConcurrencyTestSuite{})
}
