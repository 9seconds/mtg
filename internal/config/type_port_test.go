package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typePortTestStruct struct {
	Value config.TypePort `json:"value"`
}

type TypePortTestSuite struct {
	suite.Suite
}

func (suite *TypePortTestSuite) TestUnmarshalFail() {
	testData := []string{
		"",
		"port",
		"0",
		"-1",
		"1.5",
		"70000",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typePortTestStruct{}))
		})
	}
}

func (suite *TypePortTestSuite) TestUnmarshalOk() {
	testStruct := &typePortTestStruct{}
	suite.NoError(json.Unmarshal([]byte(`{"value": 5}`), testStruct))
	suite.EqualValues(5, testStruct.Value.Value)
}

func (suite *TypePortTestSuite) TestMarshalOk() {
	testStruct := &typePortTestStruct{
		Value: config.TypePort{
			Value: 10,
		},
	}

	data, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value":10}`, string(data))
}

func (suite *TypePortTestSuite) TestGet() {
	value := config.TypePort{}
	suite.EqualValues(10, value.Get(10))

	value.Value = 100
	suite.EqualValues(100, value.Get(10))
}

func TestTypePort(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypePortTestSuite{})
}
