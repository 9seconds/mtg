package network_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/9seconds/mtg/v2/network/v2"
	"github.com/stretchr/testify/suite"
)

type BaseHTTPTestSuite struct {
	suite.Suite

	http   *httptest.Server
	client *http.Client
}

func (suite *BaseHTTPTestSuite) SetupSuite() {
	suite.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(r.Header.Get("User-Agent"))) //nolint: errcheck
	}))
}

func (suite *BaseHTTPTestSuite) SetupTest() {
	suite.client = network.New(nil, "mtg/1", 0, 0, 0).MakeHTTPClient(nil)
}

func (suite *BaseHTTPTestSuite) TestGet() {
	resp, err := suite.client.Get(suite.http.URL)
	suite.NoError(err)

	defer resp.Body.Close() //nolint: errcheck

	data, err := io.ReadAll(resp.Body)
	suite.NoError(err)
	suite.Equal("mtg/1", string(data))
}

func (suite *BaseHTTPTestSuite) TearDownSuite() {
	suite.http.Close()
}

func TestBaseHTTP(t *testing.T) {
	t.Parallel()
	suite.Run(t, &BaseHTTPTestSuite{})
}
