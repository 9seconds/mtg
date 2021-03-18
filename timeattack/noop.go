package timeattack

import (
	"time"

	"github.com/9seconds/mtg/v2/mtglib"
)

type noop struct{}

func (n noop) Valid(_ time.Time) error { return nil }

func NewNoop() mtglib.TimeAttackDetector {
	return noop{}
}
