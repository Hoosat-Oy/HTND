package pebble

import (
	"os"
	"strconv"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/sstable"
)

// Options returns a pebble.Options struct optimized for Kaspa's block rate (10 blocks/s, 10,000 tx/block).
// Tuned for v2.1.0: Full Experimental struct literal to match expanded fields.
func Options() *pebble.Options {
	// Read-heavy tuning defaults (safe, conservative)
	// - Bloom filter FP target ~0.3% (12 bits per key)
	// - 8 KiB data blocks across levels for fast point lookups
	// - 4 KiB index blocks to reduce index IO on seeks

	// Bloom filters significantly cut false-positive reads on point lookups
	bloomFilterLevel := int(6)
	if v := os.Getenv("HTND_BLOOM_FILTER_LEVEL"); v != "" {
		if levl, err := strconv.Atoi(v); err == nil && levl > 0 {
			bloomFilterLevel = int(levl)
		}
	} else if v := os.Getenv("BLOOM_FILTER_LEVEL"); v != "" {
		if levl, err := strconv.Atoi(v); err == nil && levl > 0 {
			bloomFilterLevel = int(levl)
		}
	}
	bloomFilter := bloom.FilterPolicy(bloomFilterLevel)

	// Define MemTable size and thresholds. Larger memtables and higher thresholds
	// reduce flush frequency and write stalls at the cost of more peak RAM usage.
	// These are conservative for modern machines and can be adjusted via code if needed.
	memTableSize := int64(32 * 1024 * 1024) // 32 MiB (less frequent flushes)
	memTableStopWritesThreshold := 24       // allow more memtables before stalling
	baseFileSize := memTableSize * int64(memTableStopWritesThreshold)

	// Cache size: default 1 GiB, overridable via env for deployments
	// Use HTND_PEBBLE_CACHE_MB or PEBBLE_CACHE_MB if set.
	cacheBytes := int64(2 * 1024 * 1024 * 1024) // 2 GiB default
	if v := os.Getenv("HTND_PEBBLE_CACHE_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			cacheBytes = int64(mb) * 1024 * 1024
		}
	} else if v := os.Getenv("PEBBLE_CACHE_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			cacheBytes = int64(mb) * 1024 * 1024
		}
	}

	opts := &pebble.Options{
		FormatMajorVersion: 24, // v2: Modern formats; migrate DB

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

		// Cache: scale to available RAM (overridable via env)
		Cache: pebble.NewCache(cacheBytes),

		// Write/flush tuning
		MemTableSize:                uint64(memTableSize),
		MemTableStopWritesThreshold: memTableStopWritesThreshold,
		// Relax L0 thresholds so we compact less aggressively under bursts.
		// This works in tandem with larger memtables to reduce churn.
		L0CompactionThreshold: 20,
		L0StopWritesThreshold: 48,

		// v2: Dynamic compaction concurrency
		CompactionConcurrencyRange: func() (lower, upper int) { return 2, 4 },

		// WAL for durability
		DisableWAL:      false,
		WALBytesPerSync: 1 * 1024 * 1024,

		// v2: Levels as [7]LevelOptions (BlockSize, Compression func, FilterPolicy)
		Levels: [7]pebble.LevelOptions{
			// L0 [0]: Fast, no compression
			{
				BlockSize:      8 * 1024,
				IndexBlockSize: 4 * 1024,
				Compression:    func() *sstable.CompressionProfile { return sstable.NoCompression },
				FilterPolicy:   bloomFilter,
			},
			// L1 [1]: Snappy
			{
				BlockSize:      8 * 1024,
				IndexBlockSize: 4 * 1024,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomFilter,
			},
			// L2 [2]: Snappy
			{
				BlockSize:      8 * 1024,
				IndexBlockSize: 4 * 1024,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomFilter,
			},
			// L3 [3]: Snappy
			{
				BlockSize:      8 * 1024,
				IndexBlockSize: 4 * 1024,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomFilter,
			},
			// L4 [4]: Snappy
			{
				BlockSize:      8 * 1024,
				IndexBlockSize: 4 * 1024,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomFilter,
			},
			// L5 [5]: Snappy
			{
				BlockSize:      8 * 1024,
				IndexBlockSize: 4 * 1024,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomFilter,
			},
			// L6 [6]: Zstd for cold data
			{
				BlockSize:      8 * 1024, // prefer smaller blocks for point reads
				IndexBlockSize: 4 * 1024,
				Compression:    func() *sstable.CompressionProfile { return sstable.ZstdCompression },
				FilterPolicy:   bloomFilter,
			},
		},
	}

	opts.EnsureDefaults() // v2: Applies remaining defaults (e.g., IndexBlockSize)
	return opts
}
