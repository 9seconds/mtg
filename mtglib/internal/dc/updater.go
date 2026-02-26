package dc

import (
	"context"
	"sync"
	"time"
)

type updater struct {
	wg     sync.WaitGroup
	logger Logger
	period time.Duration
}

func (u *updater) Wait() {
	u.wg.Wait()
}

func (u *updater) run(ctx context.Context, callback func() error) {
	u.wg.Go(func() {
		ticker := time.NewTicker(u.period)

		defer func() {
			ticker.Stop()

			select {
			case <-ticker.C:
			default:
			}
		}()

		for {
			u.logger.Info("start update")
			if err := callback(); err != nil {
				u.logger.WarningError("cannot update", err)
			}
			u.logger.Info("updated")

			select {
			case <-ctx.Done():
				u.logger.Info("stop updating")
				return
			case <-ticker.C:
			}
		}
	})
}
