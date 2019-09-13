package antireplay

import (
	"github.com/allegro/bigcache"

	"github.com/9seconds/mtg/config"
)

var cache *bigcache.BigCache

func Add(data []byte) {
	cache.Set(string(data), nil) // nolint: errcheck
}

func Has(data []byte) bool {
	_, err := cache.Get(string(data))
	return err == nil
}

func Init() {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           1024,
		LifeWindow:       config.C.AntiReplay.EvictionTime,
		Hasher:           hasher{},
		HardMaxCacheSize: config.C.AntiReplay.MaxSize,
	})
	if err != nil {
		panic(err)
	}
	cache = c
}
