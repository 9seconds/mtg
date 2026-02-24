package dc

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PublicConfigUpdaterTestSuite struct {
	UpdaterTestSuiteBase

	u               *PublicConfigUpdater
	lock            sync.Mutex
	srv             *httptest.Server
	responseHandler func(w http.ResponseWriter)
}

func (s *PublicConfigUpdaterTestSuite) SetupSuite() {
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.lock.Lock()
		s.responseHandler(w)
		s.lock.Unlock()
	}))
}

func (s *PublicConfigUpdaterTestSuite) TearDownSuite() {
	s.srv.Close()
}

func (s *PublicConfigUpdaterTestSuite) SetupTest() {
	s.UpdaterTestSuiteBase.SetupTest()

	tg, err := New("prefer-ipv4")
	require.NoError(s.T(), err)

	s.u = NewPublicConfigUpdater(tg, s.loggerMock, s.srv.Client())
}

func (s *PublicConfigUpdaterTestSuite) Test502StatusCode() {
	s.responseHandler = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadGateway)
	}
	s.u.Run(s.ctx, s.srv.URL, "tcp4")

	time.Sleep(100 * time.Millisecond)
	s.ctxCancel()
	s.u.Wait()

	s.Len(s.u.tg.view.publicConfigs.v4, 0)
}

func (s *PublicConfigUpdaterTestSuite) TestEmptyFile() {
	s.responseHandler = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
	}
	s.u.Run(s.ctx, s.srv.URL, "tcp4")

	time.Sleep(100 * time.Millisecond)
	s.ctxCancel()
	s.u.Wait()

	s.Len(s.u.tg.view.publicConfigs.v4, 0)
}

func (s *PublicConfigUpdaterTestSuite) TestGarbage() {
	result := `
proxy_for -1 -1;
proxy_for 100 100.10.0.0:3333;
lala 0 0
`

	s.responseHandler = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result)) //nolint: errcheck
	}
	s.u.Run(s.ctx, s.srv.URL, "tcp4")

	time.Sleep(100 * time.Millisecond)
	s.ctxCancel()
	s.u.Wait()

	s.Len(s.u.tg.view.publicConfigs.v4, 0)
}

func (s *PublicConfigUpdaterTestSuite) TestOk() {
	result := `
proxy_for 203 100.10.0.0:3333;
proxy_for -100 101.10.0.0:3333;
`

	s.responseHandler = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result)) //nolint: errcheck
	}
	s.u.Run(s.ctx, s.srv.URL, "tcp4")

	time.Sleep(100 * time.Millisecond)
	s.ctxCancel()
	s.u.Wait()

	s.Len(s.u.tg.view.publicConfigs.v4, 1)
	s.Len(s.u.tg.view.publicConfigs.v4[203], 1)
	s.Equal("100.10.0.0:3333", s.u.tg.view.publicConfigs.v4[203][0].Address)
}

func TestPublicConfigUpdater(t *testing.T) {
	suite.Run(t, &PublicConfigUpdaterTestSuite{})
}
