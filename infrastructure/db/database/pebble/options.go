package pebble

import (
	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"   // ADD: For MultiLevelHeuristic if needed (or use pebble.MultiLevelHeuristic)
	"github.com/cockroachdb/pebble/v2/sstable" // For CompressionProfile
)

// Options returns a pebble.Options struct optimized for Kaspa's block rate (10 blocks/s, 10,000 tx/block).
// Tuned for v2.1.0: Full Experimental struct literal to match expanded fields.
func Options() *pebble.Options {
	// Use a Bloom filter with 10 bits per key
	bloomFilter := bloom.FilterPolicy(10)

	// Define MemTable size (balanced for bursts without overload)
	memTableSize := int64(16 * 1024 * 1024) // 16 MB
	memTableStopWritesThreshold := 12
	baseFileSize := memTableSize * int64(memTableStopWritesThreshold)

	opts := &pebble.Options{
		FormatMajorVersion: 2, // v2: Modern formats; migrate DB

		// v2: TargetFileSizes as fixed [7]int64 array (L0 [0], L6 [6])
		TargetFileSizes: [7]int64{
			baseFileSize,      // L0 [0]
			baseFileSize * 2,  // L1 [1]
			baseFileSize * 4,  // L2 [2]
			baseFileSize * 8,  // L3 [3]
			baseFileSize * 16, // L4 [4]
			baseFileSize * 32, // L5 [5]
			baseFileSize * 64, // L6 [6]
		},

		// Cache: Scale to available RAM
		Cache: pebble.NewCache(1 * 1024 * 1024 * 1024), // 1 GB

		// Write optimizations
		MemTableSize:                uint64(memTableSize),
		MemTableStopWritesThreshold: memTableStopWritesThreshold,
		L0CompactionThreshold:       12,
		L0StopWritesThreshold:       24,

		// v2: Dynamic compaction concurrency
		CompactionConcurrencyRange: func() (lower, upper int) { return 2, 4 },

		// WAL for durability
		DisableWAL:      false,
		WALBytesPerSync: 1 * 1024 * 1024,

		// v2: Levels as [7]LevelOptions (BlockSize, Compression func, FilterPolicy)
		Levels: [7]pebble.LevelOptions{
			// L0 [0]: Fast, no compression
			{
				BlockSize:    8 * 1024,
				Compression:  func() *sstable.CompressionProfile { return sstable.NoCompression },
				FilterPolicy: bloomFilter,
			},
			// L1 [1]: Snappy
			{
				BlockSize:    8 * 1024,
				Compression:  func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy: bloomFilter,
			},
			// L2 [2]: Snappy
			{
				BlockSize:    8 * 1024,
				Compression:  func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy: bloomFilter,
			},
			// L3 [3]: Snappy
			{
				BlockSize:    8 * 1024,
				Compression:  func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy: bloomFilter,
			},
			// L4 [4]: Snappy
			{
				BlockSize:    8 * 1024,
				Compression:  func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy: bloomFilter,
			},
			// L5 [5]: Snappy
			{
				BlockSize:    8 * 1024,
				Compression:  func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy: bloomFilter,
			},
			// L6 [6]: Zstd for cold data
			{
				BlockSize:    16 * 1024,
				Compression:  func() *sstable.CompressionProfile { return sstable.ZstdCompression },
				FilterPolicy: bloomFilter,
			},
		},
	}

	opts.EnsureDefaults() // v2: Applies remaining defaults (e.g., IndexBlockSize)
	return opts
}
