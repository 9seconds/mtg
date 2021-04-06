package timeattack

import (
	"fmt"
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
)

type detector struct {
	time.Duration
}

func (d detector) Valid(then time.Time) error {
	now := time.Now()

	diff := now.Sub(then)
	if diff < 0 {
		diff = -diff
	}

	if diff > d.Duration {
		return fmt.Errorf("time is invalid. now=%d, then=%d, diff=%v",
			now.Unix(),
			then.Unix(),
			diff)
	}

	return nil
}

// NewDetector returns a new TimeAttackDetector which validates that
// timestamp belongs to intervar [X-duration, X+duration], so a small
// timeshift is acceptable.
func NewDetector(duration time.Duration) mtglib.TimeAttackDetector {
	return detector{
		Duration: duration,
	}
}
