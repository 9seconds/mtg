package antireplay

import (
	"sync"

	"github.com/VictoriaMetrics/fastcache"

	"github.com/9seconds/mtg/config"
)

type CacheInterface interface {
	AddObfuscated2([]byte)
	AddTLS([]byte)
	HasObfuscated2([]byte) bool
	HasTLS([]byte) bool
}

var (
	Cache    CacheInterface
	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		if config.C.AntiReplayMaxSize == 0 {
			Cache = nilCache{}
		} else {
			Cache = cache{fastcache.New(config.C.AntiReplayMaxSize)}
		}
	})
}
