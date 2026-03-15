package doppel

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type SimpleNetwork struct{}

func (s SimpleNetwork) Dial(network, address string) (essentials.Conn, error) {
	return s.DialContext(context.Background(), network, address)
}

func (s SimpleNetwork) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	d := &net.Dialer{}

	conn, err := d.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	return conn.(*net.TCPConn), nil
}

func (s SimpleNetwork) NativeDialer() *net.Dialer {
	return &net.Dialer{}
}

func (s SimpleNetwork) MakeHTTPClient(dialFunc func(ctx context.Context, network, address string) (essentials.Conn, error)) *http.Client {
	if dialFunc == nil {
		dialFunc = s.DialContext
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint: gosec
			},
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialFunc(ctx, network, address)
			},
		},
	}
}

type TLSServerTestSuite struct {
	suite.Suite

	tlsServer *httptest.Server
	ctx       context.Context
	ctxCancel context.CancelFunc
	network   SimpleNetwork
	urls      []string
}

func (suite *TLSServerTestSuite) SetupSuite() {
	suite.tlsServer = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Hello", "how long")

		if _, err := w.Write([]byte{1, 2, 3}); err != nil {
			panic(err)
		}

		time.Sleep(5 * time.Millisecond)

		if _, err := w.Write([]byte{1, 2, 3}); err != nil {
			panic(err)
		}
	}))
	suite.urls = []string{suite.tlsServer.URL}
}

func (suite *TLSServerTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())
	suite.ctx = ctx
	suite.ctxCancel = cancel
}

func (suite *TLSServerTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.tlsServer.CloseClientConnections()
}

func (suite *TLSServerTestSuite) TearDownSuite() {
	suite.tlsServer.Close()
}

type LoggerMock struct {
	mock.Mock
}

func (l *LoggerMock) Info(msg string) {
	l.Called(msg)
}

func (l *LoggerMock) WarningError(msg string, err error) {
	l.Called(msg, err)
}
