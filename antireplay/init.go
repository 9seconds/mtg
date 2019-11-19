package antireplay

import (
	"sync"

	"github.com/VictoriaMetrics/fastcache"

	"github.com/9seconds/mtg/config"
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
