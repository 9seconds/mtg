package newantireplay

import "github.com/cespare/xxhash"

type hasher struct{}

func (h hasher) Sum64(value string) uint64 {
	return xxhash.Sum64String(value)
}
