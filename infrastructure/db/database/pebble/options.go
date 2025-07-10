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
	memTableSize := int64(64 * 1024 * 1024) // 128 Mb
	opts := &pebble.Options{
		// Increase cache size for better read performance
		Cache: pebble.NewCache(1 * 1024 * 1024 * 1024), // 1 GB

		// Write-heavy workload optimizations
		MemTableSize:                uint64(memTableSize), // 64 Mb
		MemTableStopWritesThreshold: 6,                    // 384 Mb
		L0CompactionThreshold:       16,
		L0StopWritesThreshold:       32,
		MaxConcurrentCompactions:    func() int { return 8 },

		// Configure LSM levels
		Levels: []pebble.LevelOptions{
			// Level 0: Smaller file size, Snappy for faster flushes
			{
				TargetFileSize: memTableSize, // 128 MB
				BlockSize:      16 * 1024,
				Compression:    pebble.NoCompression, // Use Snappy to reduce write amplification
				FilterPolicy:   bloomFilter,
			},
			// Level 1 to 5: Adjusted scaling
			{
				TargetFileSize: memTableSize * 2, // 256 MB
				BlockSize:      16 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 4, // 512 MB
				BlockSize:      16 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 6, // 1 GB
				BlockSize:      16 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 12, // 2 GB
				BlockSize:      16 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 24, // 4 GB
				BlockSize:      16 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			// Level 6: Cold data with Zstd
			{
				TargetFileSize: memTableSize * 48, // 8 GB
				BlockSize:      16 * 1024,
				Compression:    pebble.ZstdCompression,
				FilterPolicy:   bloomFilter,
			},
		},
	}

	// Ensure cache is properly referenced
	opts.EnsureDefaults()
	return opts
}
