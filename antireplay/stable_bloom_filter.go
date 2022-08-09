package antireplay

import (
	"sync"

	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/OneOfOne/xxhash"
	boom "github.com/tylertreat/BoomFilters"
)

type stableBloomFilter struct {
	filter boom.StableBloomFilter
	mutex  sync.Mutex
}

func (s *stableBloomFilter) SeenBefore(digest []byte) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.filter.TestAndAdd(digest)
}

// NewStableBloomFilter returns an implementation of AntiReplayCache based on
// stable bloom filter.
//
// http://webdocs.cs.ualberta.ca/~drafiei/papers/DupDet06Sigmod.pdf
//
// The basic idea of a stable bloom filter is quite simple: each time when you
// set a new element, you randomly reset P elements. There is a hardcore math
// which proves that if you choose this P correctly, you can maintain the same
// error rate for a stream of elements.
//
// byteSize is the number of bytes you want to give to a bloom filter.
// errorRate is desired false-positive error rate. If you want to use default
// values, please pass 0 for byteSize and <0 for errorRate.
func NewStableBloomFilter(byteSize uint, errorRate float64) mtglib.AntiReplayCache {
	if byteSize == 0 {
		byteSize = DefaultStableBloomFilterMaxSize
	}

	if errorRate < 0 {
		errorRate = DefaultStableBloomFilterErrorRate
	}

	sf := boom.NewDefaultStableBloomFilter(byteSize*8, errorRate) //nolint: gomnd
	sf.SetHash(xxhash.New64())

	return &stableBloomFilter{
		filter: *sf,
	}
}
