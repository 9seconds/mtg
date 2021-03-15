package antireplay

import (
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/OneOfOne/xxhash"
	boom "github.com/tylertreat/BoomFilters"
)

type stableBloomFilter struct {
	filter boom.StableBloomFilter
}

func (s *stableBloomFilter) SeenBefore(digest []byte) bool {
	return s.filter.TestAndAdd(digest)
}

func NewStableBloomFilter(byteSize uint, errorRate float64) mtglib.AntiReplayCache {
	sf := boom.NewDefaultStableBloomFilter(byteSize*8, errorRate) // nolint: gomnd
	sf.SetHash(xxhash.New64())

	return &stableBloomFilter{
		filter: *sf,
	}
}
