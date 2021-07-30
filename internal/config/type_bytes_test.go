package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeBytesTestStruct struct {
	Value config.TypeBytes `json:"value"`
}

type TypeBytesTestSuite struct {
	suite.Suite
}

func (suite *TypeBytesTestSuite) TestUnmarshalFail() {
	testData := []string{
		"1m",
		"1",
		"-1kb",
		"-1kib",
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeBytesTestStruct{}))
		})
	}
}

func (suite *TypeBytesTestSuite) TestUnmarshalOk() {
	testData := map[string]uint{
		"1b":   1,
		"1kb":  1024,
		"1kib": 1024,
		"2mb":  2 * 1024 * 1024,
		"2mib": 2 * 1024 * 1024,
	}

	for k, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": k,
		})
		suite.NoError(err)

		suite.T().Run(k, func(t *testing.T) {
			testStruct := &typeBytesTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.EqualValues(t, value, testStruct.Value.Get(0))
		})
	}
}

func (suite *TypeBytesTestSuite) TestMarshalOk() {
	value := typeBytesTestStruct{}
	suite.NoError(value.Value.Set("1kib"))

	data, err := json.Marshal(value)
	suite.NoError(err)
	suite.JSONEq(`{"value": "1kib"}`, string(data))
}

func (suite *TypeBytesTestSuite) TestGet() {
	value := config.TypeBytes{}
	suite.EqualValues(1000, value.Get(1000))

	suite.NoError(value.Set("1mib"))
	suite.EqualValues(1048576, value.Get(1000))
}

func TestTypeBytes(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeBytesTestSuite{})
}
