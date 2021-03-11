package config_test

import (
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/config"
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
		"-1QB",
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
			assert.Equal(t, int(value), int(testStruct.Value.Value(0)))
		})
	}
}

func (suite *TypeBytesTestSuite) TestMarshalOk() {
	testData := []string{
		"1b",
		"1kib",
		"2mib",
	}

	for _, v := range testData {
		name := v

		data, err := json.Marshal(map[string]string{
			"value": name,
		})
		suite.NoError(err)

		suite.T().Run(name, func(t *testing.T) {
			testStruct := &typeBytesTestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, name, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, string(marshalled), name)
		})
	}
}

func (suite *TypeBytesTestSuite) TestValue() {
	testStruct := &typeBytesTestStruct{}

	suite.EqualValues(0, testStruct.Value.Value(0))
	suite.EqualValues(1, testStruct.Value.Value(1))

	data, err := json.Marshal(map[string]string{
		"value": "1kb",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.EqualValues(1024, testStruct.Value.Value(0))
	suite.EqualValues(1024, testStruct.Value.Value(1))
}

func TestTypeBytes(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeBytesTestSuite{})
}
