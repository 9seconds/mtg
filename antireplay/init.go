package antireplay

import (
	"sync"

	"github.com/9seconds/mtg/config"
	"github.com/allegro/bigcache"
)

var (
	Cache    *cache
	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		c1, err := bigcache.NewBigCache(bigcache.Config{
			Shards:           1024,
			LifeWindow:       config.C.AntiReplayEvictionTime,
			Hasher:           hasher{},
			HardMaxCacheSize: config.C.AntiReplayMaxSize,
		})
		if err != nil {
			panic(err)
		}

		c2, err := bigcache.NewBigCache(bigcache.Config{
			Shards:           1024,
			LifeWindow:       config.C.AntiReplayEvictionTime,
			Hasher:           hasher{},
			HardMaxCacheSize: config.C.AntiReplayMaxSize,
		})
		if err != nil {
			panic(err)
		}

		Cache = &cache{
			obfuscated2: c1,
			tls:         c2,
		}
	})
}
