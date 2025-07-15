package pebble

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

// Options returns a pebble.Options struct optimized for Kaspa's block rate (10 blocks/s, 10,000 tx/block).
func Options() *pebble.Options {
	// Use a Bloom filter with 10 bits per key
	bloomFilter := bloom.FilterPolicy(10)

	// Define MemTable size
	memTableSize := int64(4 * 1024 * 1024) // 4 Mb
	memTableStopWritesThreshold := 12
	baseFileSize := memTableSize * int64(memTableStopWritesThreshold)
	opts := &pebble.Options{
		// Increase cache size for better read performance
		Cache: pebble.NewCache(1 * 1024 * 1024 * 1024), // 1 GB

		// Write-heavy workload optimizations
		MemTableSize:                uint64(memTableSize),        // 4 Mb
		MemTableStopWritesThreshold: memTableStopWritesThreshold, // 48 Mb
		L0CompactionThreshold:       4,
		L0StopWritesThreshold:       12,
		MaxConcurrentCompactions:    func() int { return 4 },

		// Configure LSM levels
		Levels: []pebble.LevelOptions{
			// Level 0: Smaller file size, Snappy for faster flushes
			{
				TargetFileSize: baseFileSize,
				BlockSize:      4 * 1024,
				Compression:    pebble.NoCompression, // Use Snappy to reduce write amplification
				FilterPolicy:   bloomFilter,
			},
			// Level 1 to 5: Adjusted scaling
			{
				TargetFileSize: baseFileSize * 2,
				BlockSize:      4 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: baseFileSize * 3,
				BlockSize:      4 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: baseFileSize * 4,
				BlockSize:      4 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: baseFileSize * 5,
				BlockSize:      4 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: baseFileSize * 6,
				BlockSize:      4 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			// Level 6: Cold data with Zstd
			{
				TargetFileSize: baseFileSize * 12,
				BlockSize:      4 * 1024,
				Compression:    pebble.ZstdCompression,
				FilterPolicy:   bloomFilter,
			},
		},
	}

	// Ensure cache is properly referenced
	opts.EnsureDefaults()
	return opts
}
