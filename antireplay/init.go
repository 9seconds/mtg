package antireplay

const (
	// DefaultStableBloomFilterMaxSize is a recommended byte size for a
	// stable bloom filter.
	DefaultStableBloomFilterMaxSize = 1024 * 1024 // 1MiB

	// DefaultStableBloomFilterErrorRate is a recommended default error
	// rate for a stable bloom filter.
	DefaultStableBloomFilterErrorRate = 0.001
)
