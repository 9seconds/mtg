package fake

import (
	"errors"
)

var ErrBadDigest = errors.New("incorrect client random")
