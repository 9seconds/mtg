package mtglib_test

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib"
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

			suite.NoError(err)
			suite.Equal(s.Key, parsed.Key)
			suite.Equal(s.Host, parsed.Host)

			newSecret := mtglib.Secret{}

			suite.NoError(newSecret.UnmarshalText([]byte(param)))

			suite.Equal(s.Key, newSecret.Key)
			suite.Equal(s.Host, newSecret.Host)
		})
	}
}

func (suite *SecretTestSuite) TestSerialize() {
	secretData, _ := hex.DecodeString("d11c6cbbd9efe7fed5bc0db220b09665")
	s := mtglib.Secret{
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
		"ed11c6cbbd9efe7fed5bc0db220b09665",
	}

	for _, v := range testData {
		param := v

		suite.T().Run(param, func(t *testing.T) {
			_, err := mtglib.ParseSecret(param)

			suite.Error(err)
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

func TestSecret(t *testing.T) {
	t.Parallel()
	suite.Run(t, &SecretTestSuite{})
}
