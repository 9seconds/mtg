package antireplay

import (
	"errors"
	"fmt"

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

func Init() error {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           1024,
		LifeWindow:       config.C.AntiReplay.EvictionTime,
		Hasher:           hasher{},
		HardMaxCacheSize: config.C.AntiReplay.MaxSize,
	})
	cache = c
	err = fmt.Errorf("qqq: %w", errors.New("tt"))

	return err
}
