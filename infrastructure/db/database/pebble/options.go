package pebble

import (
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

// Options returns a pebble.Options struct optimized for Kaspa's block rate (10 blocks/s, 10,000 tx/block)
// with WAL syncs reduced to once per second to improve write throughput.
func Options() *pebble.Options {
	// Use a Bloom filter with 10 bits per key for efficient reads
	bloomFilter := bloom.FilterPolicy(10)

	// Define MemTable size
	memTableSize := int64(64 * 1024 * 1024) // 64 MB

	opts := &pebble.Options{
		// Large block cache to optimize read performance
		Cache: pebble.NewCache(256 * 1024 * 1024), // 256 MB cache

		// Write-heavy workload optimizations
		MemTableSize:                uint64(memTableSize),
		MemTableStopWritesThreshold: 8,                       // Limit in-memory tables to prevent overload
		L0CompactionThreshold:       32,                      // Start compacting after 32 L0 files
		L0StopWritesThreshold:       64,                      // Apply backpressure after 64 L0 files
		MaxConcurrentCompactions:    func() int { return 8 }, // Allow more compactions in parallel
		DisableAutomaticCompactions: false,

		// Reduce WAL sync frequency
		WALMinSyncInterval: func() time.Duration {
			return 100 * time.Millisecond // Sync WAL at most every 100 millisecond
		},
		FlushSplitBytes: memTableSize / 2, // Split flushes to reduce WAL sync pressure

		// Configure LSM levels
		Levels: []pebble.LevelOptions{
			// Level 0: Match file size to MemTable to avoid fragmentation
			{
				TargetFileSize: memTableSize,
				BlockSize:      32 * 1024,
				Compression:    pebble.NoCompression,
				FilterPolicy:   bloomFilter,
			},
			// Level 1 to 5: Progressive scaling with Snappy compression
			{
				TargetFileSize: memTableSize * 2,
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 4,
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 8,
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 16,
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			{
				TargetFileSize: memTableSize * 32,
				BlockSize:      32 * 1024,
				Compression:    pebble.SnappyCompression,
				FilterPolicy:   bloomFilter,
			},
			// Level 6: Cold data with high compression
			{
				TargetFileSize: 2048 * 1024 * 1024,
				BlockSize:      32 * 1024,
				Compression:    pebble.ZstdCompression,
				FilterPolicy:   bloomFilter,
			},
		},
	}

	return opts
}
