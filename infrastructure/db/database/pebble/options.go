package pebble

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/bloom"
	"github.com/cockroachdb/pebble/v2/sstable"
)

// Options returns a pebble.Options struct optimized for HTND's block rate and data patterns.
//
// IMPORTANT: Keep defaults memory-safe. Pebble can reserve significant memory via the block
// cache and memtables; on Windows this may hit the system commit limit and crash with
// `VirtualAlloc ... errno=1455`.
//
// Environment variables for tuning (all optional):
//
//	HTND_BLOOM_FILTER_LEVEL - Bloom filter bits per key (default: 14 for ~0.1% false positive)
//	HTND_PEBBLE_CACHE_MB - Cache size in MB (default: caller-provided cacheSizeMiB)
//	HTND_MEMTABLE_SIZE_MB - MemTable size in MB (default: 32 MB)
//	HTND_MEMTABLE_THRESHOLD - Number of memtables before stalling writes (default: 8)
//	HTND_L0_COMPACTION_THRESHOLD - L0 compaction trigger (default: 8)
//	HTND_L0_STOP_WRITES_THRESHOLD - L0 write stall threshold (default: 48)
//	HTND_PEBBLE_LOG_EVENTS - Enable Pebble internal event logging (default: false)
//	HTND_PEBBLE_LOG_EVENTS_MIN_MS - Only log compactions/flushes >= this duration (default: 250)
//
// Legacy environment variables (for backward compatibility):
//
//	BLOOM_FILTER_LEVEL, PEBBLE_CACHE_MB
func Options(cacheSizeMiB int) *pebble.Options {
	// Note: Each increase in bloom filter level roughly halves the false positive rate:
	// - Level 10: ~1% false positive rate (10 bits per key)
	// - Level 12: ~0.4% false positive rate (12 bits per key)
	// - Level 14: ~0.1% false positive rate (14 bits per key)
	// - Level 16: ~0.025% false positive rate (16 bits per key)
	// Trade-off: Higher levels use more memory but significantly improve read performance

	// Increased default bloom filter level for better key lookup performance
	// This is especially important for virtual block hash lookups during IBD
	bloomFilterLevel := int(16)
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

	// Define MemTable size and thresholds. These must be kept conservative by default:
	// the effective peak RAM is roughly Cache + (MemTableSize * MemTableStopWritesThreshold)
	// (plus some overhead). The previous defaults could reach multiple GiB.
	memTableSize := int64(128 * 1024 * 1024) // 128 MiB
	if v := os.Getenv("HTND_MEMTABLE_SIZE_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			memTableSize = int64(mb) * 1024 * 1024
		}
	}

	memTableStopWritesThreshold := 32
	if v := os.Getenv("HTND_MEMTABLE_THRESHOLD"); v != "" {
		if threshold, err := strconv.Atoi(v); err == nil && threshold > 0 {
			memTableStopWritesThreshold = threshold
		}
	}

	baseFileSize := memTableSize * int64(memTableStopWritesThreshold)

	// Cache size: env vars override; otherwise use the caller-provided cacheSizeMiB.
	// If cacheSizeMiB is 0/negative, default to 1024 MiB.
	cacheBytes := int64(1024 * 1024 * 1024)
	if cacheSizeMiB > 0 {
		cacheBytes = int64(cacheSizeMiB) * 1024 * 1024
	}
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

		// Additional stability options
		MaxManifestFileSize:       128 * 1024 * 1024, // 128 MB manifest files
		MaxOpenFiles:              16384,             // Increased file handle limit
		L0CompactionFileThreshold: 1024,              // More aggressive L0 compaction

		// Cache: scale to available RAM (overridable via env)
		Cache: pebble.NewCache(cacheBytes),

		// Write/flush tuning
		MemTableSize:                uint64(memTableSize),
		MemTableStopWritesThreshold: memTableStopWritesThreshold,
		// More aggressive L0 thresholds for high-throughput operation
		// Prevents write stalls during sustained high TPS loads
		L0CompactionThreshold: func() int {
			if v := os.Getenv("HTND_L0_COMPACTION_THRESHOLD"); v != "" {
				if threshold, err := strconv.Atoi(v); err == nil && threshold > 0 {
					return threshold
				}
			}
			return 8
		}(),
		L0StopWritesThreshold: func() int {
			if v := os.Getenv("HTND_L0_STOP_WRITES_THRESHOLD"); v != "" {
				if threshold, err := strconv.Atoi(v); err == nil && threshold > 0 {
					return threshold
				}
			}
			return 48
		}(),

		// v2: Dynamic compaction concurrency
		CompactionConcurrencyRange: func() (lower, upper int) { return 2, 4 },

		// Enhanced WAL settings for better durability and consistency
		DisableWAL:      false,
		WALBytesPerSync: 512 * 1024,      // More frequent syncs for better consistency
		BytesPerSync:    1 * 1024 * 1024, // Regular file syncing

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

	if envBool("HTND_PEBBLE_LOG_EVENTS") {
		minDuration := 250 * time.Millisecond
		if v := os.Getenv("HTND_PEBBLE_LOG_EVENTS_MIN_MS"); v != "" {
			if ms, err := strconv.Atoi(v); err == nil && ms > 0 {
				minDuration = time.Duration(ms) * time.Millisecond
			}
		}

		opts.Logger = pebbleLoggerAdapter{}
		opts.EventListener = &pebble.EventListener{
			BackgroundError: func(err error) {
				log.Errorf("[pebble] background error: %v", err)
			},
			WriteStallBegin: func(info pebble.WriteStallBeginInfo) {
				log.Warnf("[pebble] write stall begin: %s", info.Reason)
			},
			WriteStallEnd: func() {
				log.Warnf("[pebble] write stall end")
			},
			CompactionEnd: func(info pebble.CompactionInfo) {
				if info.Err != nil {
					log.Errorf("[pebble] compaction failed job=%d reason=%s total=%s err=%v", info.JobID, info.Reason, info.TotalDuration, info.Err)
					return
				}
				if info.TotalDuration >= minDuration {
					log.Infof("[pebble] compaction job=%d reason=%s total=%s", info.JobID, info.Reason, info.TotalDuration)
				}
			},
			FlushEnd: func(info pebble.FlushInfo) {
				if info.Err != nil {
					log.Errorf("[pebble] flush failed job=%d reason=%s total=%s err=%v", info.JobID, info.Reason, info.TotalDuration, info.Err)
					return
				}
				if info.TotalDuration >= minDuration {
					log.Infof("[pebble] flush job=%d reason=%s input=%d inputBytes=%d ingest=%t total=%s",
						info.JobID, info.Reason, info.Input, info.InputBytes, info.Ingest, info.TotalDuration)
				}
			},
			DiskSlow: func(info pebble.DiskSlowInfo) {
				log.Warnf("[pebble] disk slow op=%s path=%s writeBytes=%d duration=%s", info.OpType, info.Path, info.WriteSize, info.Duration)
			},
		}
	}

	opts.EnsureDefaults() // v2: Applies remaining defaults (e.g., IndexBlockSize)
	return opts
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes" || v == "y" || v == "on"
}
