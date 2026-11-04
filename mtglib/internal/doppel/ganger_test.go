package doppel

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type GangerTestSuite struct {
	TLSServerTestSuite

	log *LoggerMock
	g   *Ganger
}

func (suite *GangerTestSuite) SetupTest() {
	suite.TLSServerTestSuite.SetupTest()

	suite.log = &LoggerMock{}
	suite.log.
		On("Info", mock.AnythingOfType("string")).
		Maybe()
	suite.log.
		On("WarningError", mock.AnythingOfType("string"), mock.Anything).
		Maybe()

	suite.g = NewGanger(suite.ctx, suite.network, suite.log, time.Hour, 1, suite.urls, true)
	suite.g.Run()
}

func (suite *GangerTestSuite) TearDownTest() {
	suite.g.Shutdown()

	suite.log.AssertExpectations(suite.T())
	suite.TLSServerTestSuite.TearDownTest()
}

func (suite *GangerTestSuite) TestNewConnAfterShutdown() {
	suite.g.Shutdown()
	connMock := &testlib.EssentialsConnMock{}

	_, err := suite.g.NewConn(connMock)
	suite.Error(err)
}

func (suite *GangerTestSuite) TestNewConnWhileRunning() {
	connMock := &testlib.EssentialsConnMock{}
	connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(0, nil).
		Maybe()
	connMock.On("Close").
		Return(nil).
		Maybe()

	conn, err := suite.g.NewConn(connMock)
	suite.NoError(err)

	conn.Stop()
}

func (suite *GangerTestSuite) TestNewConnWriteProducesTLSRecords() {
	var (
		mu  sync.Mutex
		buf bytes.Buffer
	)

	connMock := &testlib.EssentialsConnMock{}
	connMock.On("Write", mock.AnythingOfType("[]uint8")).
		Run(func(args mock.Arguments) {
			mu.Lock()
			buf.Write(args.Get(0).([]byte))
			mu.Unlock()
		}).
		Return(0, nil).
		Maybe()
	connMock.On("Close").
		Return(nil).
		Maybe()

	conn, err := suite.g.NewConn(connMock)
	suite.NoError(err)

	payload := bytes.Repeat([]byte("x"), 512)
	_, err = conn.Write(payload)
	suite.NoError(err)

	time.Sleep(500 * time.Millisecond)
	conn.Stop()

	mu.Lock()
	written := buf.Bytes()
	mu.Unlock()

	suite.NotEmpty(written)
}

func TestGanger(t *testing.T) {
	t.Parallel()

	suite.Run(t, &GangerTestSuite{})
}

func TestGangerNewConnCancelledBeforeReply(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := &Ganger{ctx: ctx, connRequests: make(chan gangerConnRequest)}
	result := make(chan error, 1)

	go func() {
		_, err := g.NewConn(nil)
		result <- err
	}()

	var req gangerConnRequest

	select {
	case req = <-g.connRequests:
	case <-time.After(time.Second):
		t.Fatal("NewConn did not submit its request")
	}

	cancel()

	select {
	case err := <-result:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("NewConn did not return after cancellation")
	}

	// The worker can still attempt a reply after the caller returns.
	select {
	case req.ret <- Conn{}:
		t.Fatal("reply was accepted after NewConn returned")
	default:
	}
}
