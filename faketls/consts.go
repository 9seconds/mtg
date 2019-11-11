package faketls

import (
	"errors"
	"time"
)

const (
	TimeSkew     = 5 * time.Second
	TimeFromBoot = 24 * 60 * 60
)

var (
	errBadDigest = errors.New("bad digest")
	errBadTime   = errors.New("bad time")

	faketlsStartBytes = [...]byte{
		0x16,
		0x03,
		0x01,
		0x02,
		0x00,
		0x01,
		0x00,
		0x01,
		0xfc,
		0x03,
		0x03,
	}
)
