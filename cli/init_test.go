package cli_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/9seconds/mtg/v2/cli"
	"github.com/9seconds/mtg/v2/mtglib/network"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type NetworkMock struct {
	mock.Mock
}

func (n *NetworkMock) Dial(network, address string) (net.Conn, error) {
	args := n.Called(network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (n *NetworkMock) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	args := n.Called(ctx, network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (n *NetworkMock) DNSResolve(network, hostname string) ([]string, error) {
	args := n.Called(network, hostname)

	return args.Get(0).([]string), args.Error(1)
}

func (n *NetworkMock) MakeHTTPClient(dialFunc network.DialFunc) *http.Client {
	return n.Called(dialFunc).Get(0).(*http.Client)
}

func (n *NetworkMock) IdleTimeout() time.Duration {
	return n.Called().Get(0).(time.Duration)
}

func (n *NetworkMock) HTTPTimeout() time.Duration {
	return n.Called().Get(0).(time.Duration)
}

type CommonTestSuite struct {
	suite.Suite

	cli         *cli.CLI
	networkMock *NetworkMock
	httpClient  *http.Client
}

func (suite *CommonTestSuite) SetupTest() {
	suite.networkMock = &NetworkMock{}
	suite.httpClient = &http.Client{}
	suite.cli = &cli.CLI{}

	httpmock.ActivateNonDefault(suite.httpClient)

	suite.networkMock.
		On("MakeHTTPClient", mock.Anything).
		Maybe().
		Return(suite.httpClient)
}

func (suite *CommonTestSuite) TearDownTest() {
	suite.networkMock.AssertExpectations(suite.T())
	httpmock.DeactivateAndReset()
}

func (suite *CommonTestSuite) CaptureStdout(callback func()) string {
	return suite.captureOutput(&os.Stdout, callback)
}

func (suite *CommonTestSuite) CaptureStderr(callback func()) string {
	return suite.captureOutput(&os.Stderr, callback)
}

func (suite *CommonTestSuite) captureOutput(filefp **os.File, callback func()) string {
	oldFp := *filefp

	defer func() {
		*filefp = oldFp
	}()

	reader, writer, _ := os.Pipe()
	buf := &bytes.Buffer{}
	closeChan := make(chan bool)

	go func() {
		io.Copy(buf, reader) // nolint: errcheck
		close(closeChan)
	}()

	*filefp = writer

	callback()

	writer.Close()
	<-closeChan

	return strings.TrimSpace(buf.String())
}
