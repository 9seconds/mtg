package antireplay

import "github.com/dgraph-io/ristretto"

var (
	prefixObfuscated2 = []byte{0x00}
	prefixTLS         = []byte{0x01}
)

type cache struct {
	data *ristretto.Cache
}

func (c *cache) AddObfuscated2(data []byte) {
	c.data.Set(keyObfuscated2(data), nil, int64(len(data)))
}

func (c *cache) AddTLS(data []byte) {
	c.data.Set(keyTLS(data), nil, int64(len(data)))
}

func (c *cache) HasObfuscated2(data []byte) bool {
	_, ok := c.data.Get(keyObfuscated2(data))
	return ok
}

func (c *cache) HasTLS(data []byte) bool {
	_, ok := c.data.Get(keyTLS(data))
	return ok
}

func keyObfuscated2(data []byte) string {
	return string(append(prefixObfuscated2, data...))
}

func keyTLS(data []byte) string {
	return string(append(prefixTLS, data...))
}
