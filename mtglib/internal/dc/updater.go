package dc

import (
	"context"
	"time"
)

type updater struct {
	logger Logger
	period time.Duration
}

func (u updater) run(ctx context.Context, callback func() error) {
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
			u.logger.WarningError("cannot update: %w", err)
		}
		u.logger.Info("updated")

		select {
		case <-ctx.Done():
			u.logger.Info("stop updating")
			return
		case <-ticker.C:
		}
	}
}
