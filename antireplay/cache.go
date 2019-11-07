package antireplay

import "github.com/allegro/bigcache"

type cache struct {
	obfuscated2 *bigcache.BigCache
	tls         *bigcache.BigCache
}

func (c *cache) AddObfuscated2(data []byte) {
	c.obfuscated2.Set(string(data), nil) // nolint: errcheck
}

func (c *cache) AddTLS(data []byte) {
	c.tls.Set(string(data), nil) // nolint: errcheck
}

func (c *cache) HasObfuscated2(data []byte) bool {
	_, err := c.obfuscated2.Get(string(data))
	return err == nil
}

func (c *cache) HasTLS(data []byte) bool {
	_, err := c.tls.Get(string(data))
	return err == nil
}
