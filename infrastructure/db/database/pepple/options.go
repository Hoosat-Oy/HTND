package pepple

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

// Options returns a pebble.Options struct optimized for Kaspa's high block rate (33 blocks/s, 10,000 tx/block).
func Options() *pebble.Options {
	// Define a Bloom filter with 10 bits per key
	bloomFilter := bloom.FilterPolicy(10)

	opts := &pebble.Options{
		// Sync settings: Balance durability and performance
		BytesPerSync:    1 * 1024 * 1024, // 1 MB to reduce sync frequency
		WALBytesPerSync: 1 * 1024 * 1024, // Sync WAL less frequently
		DisableWAL:      false,           // Ensure durability for blockchain data

		// Memory settings: Handle high write throughput
		MemTableSize:                128 * 1024 * 1024,       // 128 MB to buffer ~1-2 seconds of writes
		MemTableStopWritesThreshold: 4,                       // Allow up to 4 MemTables (512 MB total)
		MaxConcurrentCompactions:    func() int { return 4 }, // Parallel compactions for write rate

		// File settings: Support high concurrency
		MaxOpenFiles:        1000,              // Higher limit for large numbers of SSTables
		MaxManifestFileSize: 128 * 1024 * 1024, // Sufficient for high throughput

		// Cache: Improve read performance
		Cache: pebble.NewCache(512 * 1024 * 1024), // 512 MB block cache

		// LSM tree tuning: Optimize for write-heavy workload
		Levels: []pebble.LevelOptions{
			// Level 0: Frequent flushes from MemTable
			{TargetFileSize: 2 * 1024 * 1024, BlockSize: 16 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			// Level 1-5: Intermediate levels with increasing sizes
			{TargetFileSize: 8 * 1024 * 1024, BlockSize: 16 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 16 * 1024 * 1024, BlockSize: 16 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 32 * 1024 * 1024, BlockSize: 16 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 64 * 1024 * 1024, BlockSize: 16 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			{TargetFileSize: 128 * 1024 * 1024, BlockSize: 16 * 1024, Compression: pebble.SnappyCompression, FilterPolicy: bloomFilter},
			// Level 6: Largest level, optimize for storage
			{TargetFileSize: 256 * 1024 * 1024, BlockSize: 16 * 1024, Compression: pebble.ZstdCompression, FilterPolicy: bloomFilter},
		},
	}

	// Ensure the cache is properly referenced and cleaned up
	if opts.Cache != nil {
		defer opts.Cache.Unref()
	}

	return opts
}
