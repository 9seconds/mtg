package doppel

import (
	"context"
	"time"
)

type Clock struct {
	stats *Stats
	tick  chan struct{}
}

func (c Clock) Start(ctx context.Context) {
	tickTock := time.NewTimer(c.stats.Delay())
	defer func() {
		tickTock.Stop()
		select {
		case <-tickTock.C:
		default:
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickTock.C:
			select {
			case <-ctx.Done():
			case c.tick <- struct{}{}:
			}
			tickTock.Reset(c.stats.Delay())
		}
	}
}
