package antireplay

import "github.com/allegro/bigcache"

type cache struct {
	cache *bigcache.BigCache
}

func (c *cache) Add(data []byte) {
	c.cache.Set(string(data), nil) // nolint: errcheck
}

func (c *cache) Has(data []byte) bool {
	_, err := c.cache.Get(string(data))
	return err == nil
}
