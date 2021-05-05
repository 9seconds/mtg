package mtglib_test

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SecretTestSuite struct {
	suite.Suite
}

func (suite *SecretTestSuite) TestParseSecret() {
	secretData, _ := hex.DecodeString("d11c6cbbd9efe7fed5bc0db220b09665")
	s := mtglib.Secret{
		Host: "google.com",
	}

	copy(s.Key[:], secretData)

	testData := map[string]string{
		"hex":    "eed11c6cbbd9efe7fed5bc0db220b09665676f6f676c652e636f6d",
		"base64": "7tEcbLvZ7-f-1bwNsiCwlmVnb29nbGUuY29t",
	}

	for name, value := range testData {
		param := value

		suite.T().Run(name, func(t *testing.T) {
			parsed, err := mtglib.ParseSecret(param)
			assert.NoError(t, err)
			assert.Equal(t, s.Key, parsed.Key)
			assert.Equal(t, s.Host, parsed.Host)

			newSecret := mtglib.Secret{}
			assert.NoError(t, newSecret.UnmarshalText([]byte(param)))
			assert.Equal(t, s.Key, newSecret.Key)
			assert.Equal(t, s.Host, newSecret.Host)
		})
	}
}

func (suite *SecretTestSuite) TestSerialize() {
	s := mtglib.Secret{}

	data, err := s.MarshalText()
	suite.NoError(err)
	suite.Empty(data)

	secretData, _ := hex.DecodeString("d11c6cbbd9efe7fed5bc0db220b09665")
	s = mtglib.Secret{
		Host: "google.com",
	}

	copy(s.Key[:], secretData)

	suite.Equal("eed11c6cbbd9efe7fed5bc0db220b09665676f6f676c652e636f6d", s.Hex())
	suite.Equal("7tEcbLvZ7-f-1bwNsiCwlmVnb29nbGUuY29t", s.Base64())
}

func (suite *SecretTestSuite) TestMarshalData() {
	secretData, _ := hex.DecodeString("d11c6cbbd9efe7fed5bc0db220b09665")
	s := mtglib.Secret{
		Host: "google.com",
	}

	copy(s.Key[:], secretData)

	data, err := json.Marshal(&s)
	suite.NoError(err)
	suite.Equal(string(data), `"7tEcbLvZ7-f-1bwNsiCwlmVnb29nbGUuY29t"`)
}

func (suite *SecretTestSuite) TestIncorrectSecret() {
	testData := []string{
		"aaa",
		"d11c6cbbd9efe7fed5bc0db220b09665",
		"ddd11c6cbbd9efe7fed5bc0db220b09665",
		"+ueJ0q91t5XOnFYP8Xac3A",
		"eed11c6cbbd9efe7fed5bc0db220b09665",
		"eed11c6cbbd9efe7fed5bc0db220b096",
		"ed11c6cbbd9efe7fed5bc0db220b09665",
		"",
		"+**",
		"ee",
		"efd11c6cbbd9efe7fed5bc0db220b09665",
	}

	for _, v := range testData {
		param := v

		suite.T().Run(param, func(t *testing.T) {
			_, err := mtglib.ParseSecret(param)
			assert.Error(t, err)
		})
	}
}

func (suite *SecretTestSuite) TestInvariant() {
	generated := mtglib.GenerateSecret("google.com")

	parsed, err := mtglib.ParseSecret(generated.Hex())
	suite.NoError(err)
	suite.Equal(generated.Key, parsed.Key)
	suite.Equal(generated.Host, parsed.Host)
	suite.Equal("google.com", parsed.Host)
}

func (suite *SecretTestSuite) TestValid() {
	s := mtglib.Secret{}
	suite.False(s.Valid())

	s.Key[0] = 1
	suite.False(s.Valid())

	s.Host = "11"
	suite.True(s.Valid())
}

func TestSecret(t *testing.T) {
	t.Parallel()
	suite.Run(t, &SecretTestSuite{})
}
