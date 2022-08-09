// Antireplay package has cache implementations that are effective against
// replay attacks.
//
// To understand more about replay attacks, please read documentation for
// [mtglib.AntiReplayCache] interface. This package has a list of some
// implementations of this interface.
package antireplay

const (
	// DefaultStableBloomFilterMaxSize is a recommended byte size for a stable
	// bloom filter.
	DefaultStableBloomFilterMaxSize = 1024 * 1024 // 1MiB

	// DefaultStableBloomFilterErrorRate is a recommended default error rate for a
	// stable bloom filter.
	DefaultStableBloomFilterErrorRate = 0.001
)
