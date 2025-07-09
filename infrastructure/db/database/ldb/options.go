package ldb

import (
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Options returns a leveldb opt.Options struct optimized for Kaspa's high block rate (33 blocks/s, 1000 tx/block).
func Options() opt.Options {
	return opt.Options{
		Compression:            opt.SnappyCompression,     // Balances speed and storage efficiency
		NoSync:                 false,                     // Ensures data integrity for high-value blockchain data
		WriteBuffer:            64 * opt.MiB,              // Larger buffer to handle bursty writes
		BlockCacheCapacity:     1024 * opt.MiB,            // Larger cache for frequent reads
		Filter:                 filter.NewBloomFilter(10), // Bloom filter for efficient key lookups
		OpenFilesCacheCapacity: 500,                       // Higher file handle cache for concurrent access
		CompactionTableSize:    16 * opt.MiB,              // Larger SST files to reduce compaction frequency
		CompactionTotalSize:    256 * opt.MiB,             // Larger total size before compaction to reduce I/O
	}
}
