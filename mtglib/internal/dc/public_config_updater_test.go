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

	u               PublicConfigUpdater
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
	done := false

	s.responseHandler = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadGateway)
		done = true
	}
	go s.u.Run(s.ctx, s.srv.URL, "tcp4")

	s.Eventually(func() bool {
		s.lock.Lock()
		defer s.lock.Unlock()

		return done
	}, time.Second, 10*time.Millisecond)

	s.Len(s.u.tg.view.publicConfigs.v4, 0)
}

func (s *PublicConfigUpdaterTestSuite) TestEmptyFile() {
	done := false

	s.responseHandler = func(w http.ResponseWriter) {
		done = true
		w.WriteHeader(http.StatusOK)
	}
	go s.u.Run(s.ctx, s.srv.URL, "tcp4")

	s.Eventually(func() bool {
		s.lock.Lock()
		defer s.lock.Unlock()

		return done
	}, time.Second, 10*time.Millisecond)

	s.Len(s.u.tg.view.publicConfigs.v4, 0)
}

func (s *PublicConfigUpdaterTestSuite) TestGarbage() {
	result := `
proxy_for -1 -1;
proxy_for 100 100.10.0.0:3333;
lala 0 0
`
	done := false

	s.responseHandler = func(w http.ResponseWriter) {
		done = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
	go s.u.Run(s.ctx, s.srv.URL, "tcp4")

	s.Eventually(func() bool {
		s.lock.Lock()
		defer s.lock.Unlock()

		return done
	}, time.Second, 10*time.Millisecond)

	s.Len(s.u.tg.view.publicConfigs.v4, 0)
}

func (s *PublicConfigUpdaterTestSuite) TestOk() {
	result := `
proxy_for 203 100.10.0.0:3333;
proxy_for -100 101.10.0.0:3333;
`
	done := false

	s.responseHandler = func(w http.ResponseWriter) {
		done = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(result))
	}
	go s.u.Run(s.ctx, s.srv.URL, "tcp4")

	s.Eventually(func() bool {
		s.lock.Lock()
		defer s.lock.Unlock()

		return done
	}, time.Second, 10*time.Millisecond)

	s.Len(s.u.tg.view.publicConfigs.v4, 1)
	s.Len(s.u.tg.view.publicConfigs.v4[203], 1)
	s.Equal("100.10.0.0:3333", s.u.tg.view.publicConfigs.v4[203][0].Address)
}

func TestPublicConfigUpdater(t *testing.T) {
	suite.Run(t, &PublicConfigUpdaterTestSuite{})
}
