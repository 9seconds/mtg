package config_test

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type typeBlocklistURITestStruct struct {
	Value config.TypeBlocklistURI `json:"value"`
}

type TypeBlocklistURITestSuite struct {
	suite.Suite
}

func (suite *TypeBlocklistURITestSuite) TestUnmarshalNil() {
	typ := &config.TypeBlocklistURI{}
	suite.NoError(typ.UnmarshalText(nil))
	suite.Empty(typ.String())
}

func (suite *TypeBlocklistURITestSuite) TestUnknownSchema() {
	typ := &config.TypeBlocklistURI{}
    suite.Error(typ.UnmarshalText([]byte("gopher://lalala")))
}

func (suite *TypeBlocklistURITestSuite) TestEmptyHost() {
	typ := &config.TypeBlocklistURI{}
    suite.Error(typ.UnmarshalText([]byte("https:///path")))
}

func (suite *TypeBlocklistURITestSuite) TestUnmarshalFail() {
	rnd := make([]byte, 48)

	rand.Read(rnd) // nolint: errcheck

	unknownPath := base64.StdEncoding.EncodeToString(rnd)

	testData := []string{
		"1",
		unknownPath,
		"/" + unknownPath,
		"http:/",
		"gopher://lalalal",
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
	dir, _ := os.Getwd()
	dir, _ = filepath.Abs(dir)

	testData := []string{
		"http://lalala",
		filepath.Join(dir, "config.go"),
		"https://lalala",
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
			assert.EqualValues(t, value, testStruct.Value.Value(""))
		})
	}
}

func (suite *TypeBlocklistURITestSuite) TestMarshalOk() {
	dir, _ := os.Getwd()
	dir, _ = filepath.Abs(dir)

	testData := []string{
		"http://lalalal",
		filepath.Join(dir, "config.go"),
	}

	for _, v := range testData {
		name := v

		data, err := json.Marshal(map[string]string{
			"value": name,
		})
		suite.NoError(err)

		suite.T().Run(name, func(t *testing.T) {
			testStruct := &typeBlocklistURITestStruct{}

			assert.NoError(t, json.Unmarshal(data, testStruct))
			assert.Equal(t, name, testStruct.Value.String())

			marshalled, err := testStruct.Value.MarshalText()
			assert.NoError(t, err)
			assert.Equal(t, name, string(marshalled))
		})
	}
}

func (suite *TypeBlocklistURITestSuite) TestValue() {
	testStruct := &typeBlocklistURITestStruct{}

	suite.Equal("http://lalala", testStruct.Value.Value("http://lalala"))

	data, err := json.Marshal(map[string]string{
		"value": "http://blablabla",
	})
	suite.NoError(err)
	suite.NoError(json.Unmarshal(data, testStruct))

	suite.Equal("http://blablabla", testStruct.Value.Value(""))
}

func (suite *TypeBlocklistURITestSuite) TestIsRemote() {
	dir, _ := os.Getwd()
	dir, _ = filepath.Abs(dir)

	testData := map[bool]string{
		true:  "http://lalalal",
		false: filepath.Join(dir, "config.go"),
	}

	for k, v := range testData {
		ok := k

		data, err := json.Marshal(map[string]string{
			"value": v,
		})
		suite.NoError(err)

		suite.T().Run(strconv.FormatBool(ok), func(t *testing.T) {
			testStruct := &typeBlocklistURITestStruct{}
			assert.NoError(t, json.Unmarshal(data, testStruct))

			if ok {
				assert.True(t, testStruct.Value.IsRemote())
			} else {
				assert.False(t, testStruct.Value.IsRemote())
			}
		})
	}
}

func TestTypeBlocklistURI(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeBlocklistURITestSuite{})
}
