package antireplay

import (
	"github.com/allegro/bigcache"
	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
)

// Cache defines storage for obfuscated2 handshake frames.
type Cache struct {
	cache *bigcache.BigCache
}

func (a Cache) Add(frame []byte) {
	a.cache.Set(string(frame), nil) // nolint: errcheck
}

func (a Cache) Has(frame []byte) bool {
	_, err := a.cache.Get(string(frame))

	return err == nil
}

func NewCache(config *config.Config) (Cache, error) {
	cache, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           1024,
		LifeWindow:       config.AntiReplayEvictionTime,
		Hasher:           hasher{},
		HardMaxCacheSize: config.AntiReplayMaxSize,
	})
	if err != nil {
		return Cache{}, errors.Annotate(err, "Cannot make cache")
	}

	return Cache{cache}, nil
}
