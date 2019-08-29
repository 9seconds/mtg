package newantireplay

import (
	"github.com/allegro/bigcache"

	"github.com/9seconds/mtg/newconfig"
)

var cache *bigcache.BigCache

func Add(data []byte) {
	cache.Set(string(data), nil)
}

func Has(data []byte) bool {
	_, err := cache.Get(string(data))
	return err == nil
}

func Init() {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           1024,
		LifeWindow:       newconfig.C.AntiReplay.EvictionTime,
		Hasher:           hasher{},
		HardMaxCacheSize: newconfig.C.AntiReplay.MaxSize,
	})
	if err != nil {
		panic(err)
	}

	cache = c
}
