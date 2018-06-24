package proxy

import (
	"sync"

	"github.com/9seconds/mtg/config"
)

var copyPool sync.Pool

func init() {
	copyPool = sync.Pool{
		New: func() interface{} {
			data := make([]byte, config.BufferSizeCopy)
			return &data
		},
	}
}
