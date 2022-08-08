package network

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type CircuitBreakerTestSuite struct {
	suite.Suite

	d              Dialer
	mutex          sync.Mutex
	ctx            context.Context
	ctxCancel      context.CancelFunc
	connMock       *testlib.EssentialsConnMock
	baseDialerMock *DialerMock
}

func (suite *CircuitBreakerTestSuite) SetupTest() {
	suite.mutex = sync.Mutex{}
	suite.ctx, suite.ctxCancel = context.WithCancel(context.Background())
	suite.baseDialerMock = &DialerMock{}
	suite.connMock = &testlib.EssentialsConnMock{}
	suite.d = newCircuitBreakerDialer(suite.baseDialerMock,
		3, 100*time.Millisecond, 50*time.Millisecond)
}

func (suite *CircuitBreakerTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.baseDialerMock.AssertExpectations(suite.T())
	suite.connMock.AssertExpectations(suite.T())
}

func (suite *CircuitBreakerTestSuite) TestMultipleRunsOk() {
	suite.connMock.On("RemoteAddr").
		Times(5).
		Return(&net.TCPAddr{
			IP:   net.ParseIP("127.0.0.1"),
			Port: 3128,
		})
	suite.baseDialerMock.On("DialContext", mock.Anything, "tcp", "127.0.0.1").
		Times(5).
		Return(suite.connMock, nil)

	wg := &sync.WaitGroup{}
	wg.Add(5)

	go func() {
		wg.Wait()
		suite.ctxCancel()
	}()

	for i := 0; i < 5; i++ {
		go func() {
			defer wg.Done()

			conn, err := suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1")

			suite.mutex.Lock()
			defer suite.mutex.Unlock()

			suite.NoError(err)
			suite.Equal("127.0.0.1:3128", conn.RemoteAddr().String())
		}()
	}

	suite.Eventually(func() bool {
		_, ok := <-suite.ctx.Done()

		return !ok
	}, time.Second, 10*time.Millisecond)
}

func (suite *CircuitBreakerTestSuite) TestFromClosedToOpen() {
	suite.baseDialerMock.On("DialContext", mock.Anything, "tcp", "127.0.0.1").
		Times(3).
		Return(&net.TCPConn{}, io.EOF)

	_, err := suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1")
	suite.True(errors.Is(err, io.EOF))

	_, err = suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1")
	suite.True(errors.Is(err, io.EOF))

	_, err = suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1")
	suite.True(errors.Is(err, io.EOF))

	_, err = suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1")
	suite.True(errors.Is(err, ErrCircuitBreakerOpened))
}

func (suite *CircuitBreakerTestSuite) TestHalfOpen() {
	suite.baseDialerMock.On("DialContext", mock.Anything, "tcp", "127.0.0.1").
		Times(4).
		Return(&net.TCPConn{}, io.EOF)
	suite.baseDialerMock.On("DialContext", mock.Anything, "tcp", "127.0.0.2").
		Twice().
		Return(suite.connMock, nil)
	suite.connMock.On("RemoteAddr").Return(&net.TCPAddr{
		IP:   net.ParseIP("10.0.0.10"),
		Port: 80,
	})

	suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1") //nolint: errcheck
	suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1") //nolint: errcheck
	suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1") //nolint: errcheck
	suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1") //nolint: errcheck

	time.Sleep(500 * time.Millisecond)

	_, err := suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1")
	suite.True(errors.Is(err, io.EOF))

	_, err = suite.d.DialContext(suite.ctx, "tcp", "127.0.0.1")
	suite.True(errors.Is(err, ErrCircuitBreakerOpened))

	time.Sleep(500 * time.Millisecond)

	conn, err := suite.d.DialContext(suite.ctx, "tcp", "127.0.0.2")
	suite.NoError(err)
	suite.Equal("10.0.0.10:80", conn.RemoteAddr().String())

	_, err = suite.d.DialContext(suite.ctx, "tcp", "127.0.0.2")
	suite.NoError(err)
}

func TestCircuitBreaker(t *testing.T) {
	t.Parallel()
	suite.Run(t, &CircuitBreakerTestSuite{})
}
