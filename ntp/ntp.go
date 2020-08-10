package ntp

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/9seconds/mtg/config"
	"github.com/beevik/ntp"
	"go.uber.org/zap"
)

const autoUpdatePeriod = time.Minute

// Fetch fetches the data on time drift.
func Fetch() (time.Duration, error) {
	url := config.C.NTPServers[rand.Intn(len(config.C.NTPServers))] // nolint: gosec

	resp, err := ntp.Query(url)
	if err != nil {
		return 0, fmt.Errorf("cannot fetch NTP server %s: %w", url, err)
	}

	offsetInt := int64(resp.ClockOffset)
	if offsetInt < 0 {
		offsetInt = -offsetInt
	}

	offset := time.Duration(offsetInt)

	return offset, nil
}

// AutoUpdate runs periodic check of current time .drift state.
func AutoUpdate() {
	logger := zap.S().Named("ntp")

	for range time.Tick(autoUpdatePeriod) {
		diff, err := Fetch()
		if err != nil {
			logger.Debugw("Cannot fetch time from NTP", "error", err)

			continue
		}

		switch {
		case diff < 400*time.Millisecond:
			logger.Debugw("NTP time drift", "value", diff.String())
		case diff < 600*time.Millisecond:
			logger.Infow("NTP time drift", "value", diff.String())
		case diff < 800*time.Millisecond:
			logger.Warnw("NTP time drift", "value", diff.String())
		default:
			logger.Errorw("NTP time drift", "value", diff.String())
		}
	}
}
