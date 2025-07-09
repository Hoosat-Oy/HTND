package pepple

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

// Options returns a pebble.Options struct optimized for Kaspa's high block rate (33 blocks/s, 10,000 tx/block).
func Options() *pebble.Options {
	// Define a Bloom filter with 8 bits per key for space efficiency
	bloomFilter := bloom.FilterPolicy(10)

	opts := &pebble.Options{
		// Cache: Improve read performance for state queries
		Cache: pebble.NewCache(2 * 1024 * 1024 * 1024), // 2 GB block cache

		// LSM tree tuning: Optimize for write-heavy workload
		Levels: []pebble.LevelOptions{
			// Level 0: Frequent flushes from MemTable
			{TargetFileSize: 8 * 1024 * 1024, BlockSize: 32 * 1024, Compression: pebble.NoCompression, FilterPolicy: bloomFilter},
			// Level 1-5: Intermediate levels with increasing sizes
			{TargetFileSize: 16 * 1024 * 1024, BlockSize: 32 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 32 * 1024 * 1024, BlockSize: 32 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 64 * 1024 * 1024, BlockSize: 32 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 128 * 1024 * 1024, BlockSize: 32 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 256 * 1024 * 1024, BlockSize: 32 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			// Level 6: Largest level, optimize for storage
			{TargetFileSize: 512 * 1024 * 1024, BlockSize: 32 * 1024, Compression: pebble.ZstdCompression, FilterPolicy: bloomFilter},
		},
	}
	return opts
}
