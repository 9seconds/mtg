package antireplay

import (
	"sync"

	"github.com/VictoriaMetrics/fastcache"

	"mtg/config"
)

var (
	Cache    cache
	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		Cache.data = fastcache.New(config.C.AntiReplayMaxSize)
	})
}
