package dc

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type UpdaterTestSuite struct {
	UpdaterTestSuiteBase

	u updater
}

func (s *UpdaterTestSuite) SetupTest() {
	s.UpdaterTestSuiteBase.SetupTest()
	s.u = updater{
		logger: s.loggerMock,
		period: 100 * time.Millisecond,
	}
}

func (s *UpdaterTestSuite) TestPeriodicUpdates() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	lock := &sync.Mutex{}
	collected := []time.Time{}

	go s.u.run(s.ctx, func() error {
		select {
		case <-s.ctx.Done():
		case value := <-ticker.C:
			lock.Lock()
			collected = append(collected, value)
			lock.Unlock()
		}

		return nil
	})

	s.Eventually(func() bool {
		lock.Lock()
		defer lock.Unlock()

		return len(collected) == 3
	}, time.Second, 10*time.Millisecond)
}

func TestUpdater(t *testing.T) {
	t.Parallel()
	suite.Run(t, &UpdaterTestSuite{})
}
