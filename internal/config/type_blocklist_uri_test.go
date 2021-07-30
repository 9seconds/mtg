package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeBlocklistURITestStruct struct {
	Value config.TypeBlocklistURI `json:"value"`
}

type TypeBlocklistURITestSuite struct {
	suite.Suite

	directory    string
	absDirectory string
}

func (suite *TypeBlocklistURITestSuite) SetupSuite() {
	dir, _ := os.Getwd()
	absDir, _ := filepath.Abs(dir)

	suite.directory = dir
	suite.absDirectory = absDir
}

func (suite *TypeBlocklistURITestSuite) TestUnmarshalFail() {
	testData := []string{
		"gopher://lalala",
		"https:///paths",
		"h:/=",
		filepath.Join(suite.directory, "___"),
		filepath.Join(suite.absDirectory, "___"),
		suite.directory,
		suite.absDirectory,
	}

	for _, v := range testData {
		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			assert.Error(t, json.Unmarshal(data, &typeBlocklistURITestStruct{}))
		})
	}
}

func (suite *TypeBlocklistURITestSuite) TestUnmarshalOk() {
	testData := []string{
		"http://lalala",
		"https://lalala",
		"https://lalala/path",
		filepath.Join(suite.directory, "config.go"),
		filepath.Join(suite.absDirectory, "config.go"),
	}

	for _, v := range testData {
		value := v

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(v, func(t *testing.T) {
			testStruct := &typeBlocklistURITestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.EqualValues(t, value, testStruct.Value.Get(""))

			if strings.HasPrefix(value, "http") {
				assert.True(t, testStruct.Value.IsRemote())
			} else {
				assert.False(t, testStruct.Value.IsRemote())
			}
		})
	}
}

func (suite *TypeBlocklistURITestSuite) TestMarshalOk() {
	testStruct := &typeBlocklistURITestStruct{
		Value: config.TypeBlocklistURI{
			Value: "http://some.url/with/path",
		},
	}

	data, err := json.Marshal(testStruct)
	suite.NoError(err)
	suite.JSONEq(`{"value": "http://some.url/with/path"}`, string(data))
}

func (suite *TypeBlocklistURITestSuite) TestGet() {
	value := config.TypeBlocklistURI{}
	suite.Equal("/path", value.Get("/path"))

	suite.NoError(value.Set("http://lalala.ru"))
	suite.Equal("http://lalala.ru", value.Get("/path"))
	suite.Equal("http://lalala.ru", value.Get(""))
}

func TestTypeBlocklistURI(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeBlocklistURITestSuite{})
}
