package ipblocklist_test

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/network"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type FireholTestSuite struct {
	suite.Suite

	networkMock *testlib.MtglibNetworkMock
	httpServer  *httptest.Server
}

func (suite *FireholTestSuite) SetupSuite() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		filefp, err := os.Open(filepath.Join("testdata", "remote_ipset.ipset"))
		if err != nil {
			panic(err)
		}

		defer filefp.Close()

		io.Copy(w, filefp) //nolint: errcheck
	})

	suite.httpServer = httptest.NewServer(mux)
}

func (suite *FireholTestSuite) SetupTest() {
	httpClient := &http.Client{}
	suite.networkMock = &testlib.MtglibNetworkMock{}

	httpmock.ActivateNonDefault(httpClient)

	suite.networkMock.
		On("MakeHTTPClient", mock.Anything).
		Maybe().
		Return(httpClient)
}

func (suite *FireholTestSuite) TearDownTest() {
	suite.networkMock.AssertExpectations(suite.T())
	httpmock.DeactivateAndReset()
}

func (suite *FireholTestSuite) TearDownSuite() {
	suite.httpServer.Close()
}

func (suite *FireholTestSuite) TestLocalFail() {
	blocklist, err := ipblocklist.NewFirehol(logger.NewNoopLogger(),
		suite.networkMock, 2,
		nil, []string{filepath.Join("testdata", "broken_ipset.ipset")},
		nil)

	suite.NoError(err)

	go blocklist.Run(time.Hour)

	time.Sleep(500 * time.Millisecond)

	suite.False(blocklist.Contains(net.ParseIP("10.0.0.10")))
	suite.False(blocklist.Contains(net.ParseIP("127.0.0.1")))

	blocklist.Shutdown()
	time.Sleep(500 * time.Millisecond)
}

func (suite *FireholTestSuite) TestLocalOk() {
	blocklist, err := ipblocklist.NewFirehol(logger.NewNoopLogger(),
		suite.networkMock, 2,
		nil, []string{filepath.Join("testdata", "good_ipset.ipset")},
		nil)

	suite.NoError(err)

	go blocklist.Run(time.Hour)

	time.Sleep(500 * time.Millisecond)

	suite.True(blocklist.Contains(net.ParseIP("10.0.0.10")))
	suite.False(blocklist.Contains(net.ParseIP("127.0.0.1")))

	blocklist.Shutdown()
	time.Sleep(500 * time.Millisecond)
}

func (suite *FireholTestSuite) TestRemoteFail() {
	blocklist, err := ipblocklist.NewFirehol(logger.NewNoopLogger(),
		suite.networkMock, 2,
		[]string{"https://google.com"}, nil, nil)

	suite.NoError(err)

	go blocklist.Run(time.Hour)

	time.Sleep(500 * time.Millisecond)

	suite.False(blocklist.Contains(net.ParseIP("10.2.2.2")))

	blocklist.Shutdown()
	time.Sleep(500 * time.Millisecond)
}

func (suite *FireholTestSuite) TestMixed() {
	dialer, _ := network.NewDefaultDialer(0, 0)
	ntw, _ := network.NewNetwork(dialer, "mtg", "1.1.1.1", 0)

	blocklist, err := ipblocklist.NewFirehol(logger.NewNoopLogger(),
		ntw, 2,
		[]string{
			suite.httpServer.URL,
		}, []string{
			filepath.Join("testdata", "good_ipset.ipset"),
		}, nil)

	suite.NoError(err)

	go blocklist.Run(time.Hour)

	time.Sleep(500 * time.Millisecond)

	suite.True(blocklist.Contains(net.ParseIP("10.2.2.2")))
	suite.True(blocklist.Contains(net.ParseIP("10.1.0.100")))

	blocklist.Shutdown()
	time.Sleep(500 * time.Millisecond)
}

func TestFirehol(t *testing.T) {
	t.Parallel()
	suite.Run(t, &FireholTestSuite{})
}
