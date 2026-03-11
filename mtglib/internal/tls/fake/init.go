package fake

import (
	"errors"
	"time"
)

const (
	ClientHelloReadTimeout = 5 * time.Second
)

var (
	resetDeadline time.Time

	ErrBadDigest = errors.New("incorrect client random")
)
