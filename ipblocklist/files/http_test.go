package files_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/stretchr/testify/suite"
)

type HTTPTestSuite struct {
	suite.Suite

	httpClient *http.Client
	httpServer *httptest.Server
	ctx        context.Context
	ctxCancel  context.CancelFunc
}

func (suite *HTTPTestSuite) makeFile(path string) (files.File, error) {
	return files.NewHTTP(suite.httpClient, suite.httpServer.URL+"/"+path) //nolint: wrapcheck
}

func (suite *HTTPTestSuite) SetupSuite() {
	mux := http.NewServeMux()

	mux.Handle("/", http.FileServer(http.Dir("testdata")))

	suite.httpServer = httptest.NewServer(mux)
	suite.httpClient = suite.httpServer.Client()
}

func (suite *HTTPTestSuite) SetupTest() {
	suite.ctx, suite.ctxCancel = context.WithCancel(context.Background())
}

func (suite *HTTPTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.httpServer.CloseClientConnections()
}

func (suite *HTTPTestSuite) TearDownSuite() {
	suite.httpServer.Close()
}

func (suite *HTTPTestSuite) TestBadURL() {
	_, err := files.NewHTTP(suite.httpClient, "sdfsdf")
	suite.Error(err)
}

func (suite *HTTPTestSuite) TestBadSchema() {
	_, err := files.NewHTTP(suite.httpClient, "gopher://lala")
	suite.Error(err)
}

func (suite *HTTPTestSuite) TestNilHTTPClient() {
	_, err := files.NewHTTP(nil, "")
	suite.Error(err)
}

func (suite *HTTPTestSuite) TestAbsentFile() {
	file, err := suite.makeFile("absent")
	suite.NoError(err)

	_, err = file.Open(suite.ctx)
	suite.Error(err)
}

func (suite *HTTPTestSuite) TestOk() {
	file, err := suite.makeFile("readable")
	suite.NoError(err)

	readCloser, err := file.Open(suite.ctx)
	suite.NoError(err)

	defer readCloser.Close()

	data, err := io.ReadAll(readCloser)
	suite.NoError(err)
	suite.Equal("Hooray!", strings.TrimSpace(string(data)))
}

func TestHTTP(t *testing.T) {
	t.Parallel()
	suite.Run(t, &HTTPTestSuite{})
}
