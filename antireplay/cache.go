package antireplay

import "github.com/VictoriaMetrics/fastcache"

var (
	prefixObfuscated2 = []byte{0x00}
	prefixTLS         = []byte{0x01}
)

type cache struct {
	data *fastcache.Cache
}

func (c *cache) AddObfuscated2(data []byte) {
	c.data.Set(keyObfuscated2(data), nil)
}

func (c *cache) AddTLS(data []byte) {
	c.data.Set(keyTLS(data), nil)
}

func (c *cache) HasObfuscated2(data []byte) bool {
	return c.data.Has(keyObfuscated2(data))
}

func (c *cache) HasTLS(data []byte) bool {
	return c.data.Has(keyTLS(data))
}

func keyObfuscated2(data []byte) []byte {
	return append(prefixObfuscated2, data...)
}

func keyTLS(data []byte) []byte {
	return append(prefixTLS, data...)
}
