package network_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/network"
	"github.com/stretchr/testify/suite"
)

type NetworkTestSuite struct {
	suite.Suite
	HTTPServerTestSuite

	dialer network.Dialer
}

func (suite *NetworkTestSuite) SetupTest() {
	dialer, err := network.NewDefaultDialer(0, 0)
	suite.NoError(err)

	suite.dialer = dialer
}

func (suite *NetworkTestSuite) TestLocalHTTPRequest() {
	ntw, err := network.NewNetwork(suite.dialer, "itsme", "1.1.1.1", 0)
	suite.NoError(err)

	client := ntw.MakeHTTPClient(nil)

	resp, err := client.Get(suite.httpServer.URL + "/headers") //nolint: noctx
	suite.NoError(err)

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	jsonStruct := struct {
		Headers struct {
			UserAgent []string `json:"User-Agent"` //nolint: tagliatelle
		} `json:"headers"`
	}{}

	suite.NoError(json.Unmarshal(data, &jsonStruct))
	suite.Equal([]string{"itsme"}, jsonStruct.Headers.UserAgent)
}

func (suite *NetworkTestSuite) TestRealHTTPRequest() {
	ntw, err := network.NewNetwork(suite.dialer, "itsme", "1.1.1.1", 0)
	suite.NoError(err)

	client := ntw.MakeHTTPClient(nil)

	resp, err := client.Get("https://httpbin.org/headers") //nolint: noctx
	suite.NoError(err)

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)

	jsonStruct := struct {
		Headers struct {
			UserAgent string `json:"User-Agent"` //nolint: tagliatelle
		} `json:"headers"`
	}{}

	suite.NoError(json.Unmarshal(data, &jsonStruct))
	suite.Equal("itsme", jsonStruct.Headers.UserAgent)
}

func (suite *NetworkTestSuite) TestIncorrectTimeout() {
	_, err := network.NewNetwork(suite.dialer, "itsme", "1.1.1.1", -time.Second)
	suite.Error(err)
}

func (suite *NetworkTestSuite) TestIncorrectDOHHostname() {
	_, err := network.NewNetwork(suite.dialer, "itsme", "doh.com", 0)
	suite.Error(err)
}

func TestNetwork(t *testing.T) {
	t.Parallel()
	suite.Run(t, &NetworkTestSuite{})
}
