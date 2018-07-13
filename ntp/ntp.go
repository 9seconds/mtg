package ntp

import (
	"math/rand"
	"time"

	"github.com/beevik/ntp"
	"github.com/juju/errors"
	"go.uber.org/zap"
)

const autoUpdatePeriod = time.Minute

var ntpEndpoints = []string{
	"0.pool.ntp.org",
	"1.pool.ntp.org",
	"2.pool.ntp.org",
	"3.pool.ntp.org",
}

// Fetch fetches the data on time drift.
func Fetch() (time.Duration, error) {
	url := ntpEndpoints[rand.Intn(len(ntpEndpoints))]
	resp, err := ntp.Query(url)
	if err != nil {
		return 0, errors.Annotatef(err, "Cannot fetch NTP server %s", url)
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
