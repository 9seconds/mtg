package cli_test

import (
	"net"
	"net/http"
	"testing"

	"github.com/9seconds/mtg/v2/config"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/suite"
	"github.com/xeipuuv/gojsonschema"
)

var accressResponseJSONSchema = func() *gojsonschema.Schema {
	schema, err := gojsonschema.NewSchema(gojsonschema.NewStringLoader(`
{
    "type": "object",
    "required": ["secret"],
    "additionalProperties": true,
    "properties": {
        "secret": {
            "type": "object",
            "required": [
                "hex",
                "base64"
            ],
            "additionalProperties": false,
            "properties": {
                "hex": {
                    "type": "string",
                    "minLength": 34
                },
                "base64": {
                    "type": "string",
                    "minLength": 10
                }
            }
        },
        "ipv4": {
            "$ref": "#/definitions/ip"
        },
        "ipv6": {
            "$ref": "#/definitions/ip"
        }
    },
    "definitions": {
        "ip": {
            "type": "object",
            "required": [
                "ip",
                "port",
                "tg_url",
                "tg_qrcode",
                "tme_url",
                "tme_qrcode"
            ],
            "additionalProperties": false,
            "properties": {
                "ip": {
                    "type": "string",
                    "minLength": 1,
                    "anyOf": [
                        {
                            "format": "ipv4"
                        },
                        {
                            "format": "ipv6"
                        }
                    ]
                },
                "port": {
                    "type": "integer",
                    "multipleOf": 1.0,
                    "exclusiveMinimum": 0,
                    "exclusiveMaximum": 65536
                },
                "tg_url": {
                    "type": "string",
                    "minLength": 1,
                    "format": "uri"
                },
                "tg_qrcode": {
                    "type": "string",
                    "minLength": 1,
                    "format": "uri"
                },
                "tme_url": {
                    "type": "string",
                    "minLength": 1,
                    "format": "uri"
                },
                "tme_qrcode": {
                    "type": "string",
                    "minLength": 1,
                    "format": "uri"
                }
            }
        }
    }
}
    `))
	if err != nil {
		panic(err)
	}

	return schema
}()

type AccessTestSuite struct {
	CommonTestSuite
}

func (suite *AccessTestSuite) SetupTest() {
	suite.CommonTestSuite.SetupTest()

	suite.cli.Access.Config = &config.Config{}
	suite.cli.Access.Config.Secret = mtglib.GenerateSecret("google.com")
	suite.cli.Access.Network = suite.networkMock

	suite.NoError(
		suite.cli.Access.Config.BindTo.UnmarshalText([]byte("0.0.0.0:80")))
}

func (suite *AccessTestSuite) TestGenerateNoCalls() {
	suite.cli.Access.PublicIPv4 = net.ParseIP("10.0.0.10")
	suite.cli.Access.PublicIPv6 = net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")

	output := suite.CaptureStdout(func() {
		suite.NoError(suite.cli.Access.Execute(suite.cli))
	})

	validated, err := accressResponseJSONSchema.Validate(
		gojsonschema.NewStringLoader(output))
	suite.NoError(err)
	suite.Empty(validated.Errors())
	suite.True(validated.Valid())

	suite.Contains(output, "10.0.0.10")
	suite.Contains(output, "2001:db8:85a3::8a2e:370:7334")
	suite.Contains(output, "ipv4")
	suite.Contains(output, "ipv6")
	suite.Contains(output, suite.cli.Access.Config.Secret.Base64())
	suite.Contains(output, suite.cli.Access.Config.Secret.Hex())
}

func (suite *AccessTestSuite) TestGenerateIPv4Call() {
	suite.cli.Access.PublicIPv6 = net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")

	httpmock.RegisterResponder(http.MethodGet, "https://ifconfig.co",
		httpmock.NewStringResponder(http.StatusOK, "10.11.12.13"))

	output := suite.CaptureStdout(func() {
		suite.NoError(suite.cli.Access.Execute(suite.cli))
	})

	validated, err := accressResponseJSONSchema.Validate(
		gojsonschema.NewStringLoader(output))
	suite.NoError(err)
	suite.Empty(validated.Errors())
	suite.True(validated.Valid())

	suite.Contains(output, "10.11.12.13")
	suite.Contains(output, "2001:db8:85a3::8a2e:370:7334")
	suite.Contains(output, "ipv4")
	suite.Contains(output, "ipv6")
	suite.Contains(output, suite.cli.Access.Config.Secret.Base64())
	suite.Contains(output, suite.cli.Access.Config.Secret.Hex())
}

func (suite *AccessTestSuite) TestIPv4CallFail() {
	suite.cli.Access.PublicIPv6 = net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334")

	httpmock.RegisterResponder(http.MethodGet, "https://ifconfig.co",
		httpmock.NewStringResponder(http.StatusForbidden, ""))

	output := suite.CaptureStdout(func() {
		suite.NoError(suite.cli.Access.Execute(suite.cli))
	})

	validated, err := accressResponseJSONSchema.Validate(
		gojsonschema.NewStringLoader(output))
	suite.NoError(err)
	suite.Empty(validated.Errors())
	suite.True(validated.Valid())

	suite.Contains(output, "2001:db8:85a3::8a2e:370:7334")
	suite.NotContains(output, "ipv4")
	suite.Contains(output, "ipv6")
	suite.Contains(output, suite.cli.Access.Config.Secret.Base64())
	suite.Contains(output, suite.cli.Access.Config.Secret.Hex())
}

func TestAccess(t *testing.T) {
	t.Parallel()
	suite.Run(t, &AccessTestSuite{})
}
