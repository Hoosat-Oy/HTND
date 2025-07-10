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
	memTableSize := int64(256 * 1024 * 1024) // 256 MB (reduced to increase flush frequency)

	opts := &pebble.Options{
		// Increase cache size for better read performance
		Cache: pebble.NewCache(4 * 1024 * 1024 * 1024), // 4 GB cache

		// Write-heavy workload optimizations
		MemTableSize:                uint64(memTableSize),
		MemTableStopWritesThreshold: 12,                       // Allow more MemTables to avoid stalls
		L0CompactionThreshold:       32,                       // Compact earlier to reduce read amplification
		L0StopWritesThreshold:       64,                       // Apply backpressure earlier to control L0 growth
		MaxConcurrentCompactions:    func() int { return 12 }, // Reduce to balance I/O and CPU usage

		// Configure LSM levels
		Levels: []pebble.LevelOptions{
			// Level 0: Smaller file size, Snappy for faster flushes
			{
				TargetFileSize: memTableSize / 2, // 128 MB
				BlockSize:      32 * 1024,
				Compression:    pebble.NoCompression, // Use Snappy to reduce write amplification
				FilterPolicy:   bloomFilter,
			},
			// Level 1 to 5: Adjusted scaling
			{
				TargetFileSize: memTableSize, // 256 MB
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 2, // 512 MB
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 4, // 1 GB
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 8, // 2 GB
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 16, // 4 GB
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			// Level 6: Cold data with Zstd
			{
				TargetFileSize: memTableSize * 16, // 4 GB
				BlockSize:      32 * 1024,
				Compression:    pebble.ZstdCompression,
				FilterPolicy:   bloomFilter,
			},
		},
	}

	// Ensure cache is properly referenced
	opts.EnsureDefaults()
	return opts
}
