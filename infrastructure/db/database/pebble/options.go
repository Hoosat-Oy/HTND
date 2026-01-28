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

// Options returns Pebble configuration tuned for HTND's workload:
// high block rate, frequent point lookups, sustained write throughput.
//
// Defaults are kept memory-safe (important especially on Windows).
func Options(cacheSizeMiB int) *pebble.Options {
	// ────────────────────────────────────────────────
	// Bloom filter (critical for point lookup performance)
	// ────────────────────────────────────────────────
	// Higher bits → fewer false positives → faster reads
	//   10 ≈ 1.0%, 12 ≈ 0.4%, 14 ≈ 0.1%, 16 ≈ 0.025%
	bloomBitsPerKey := 16 // default: aggressive for IBD / hash lookups

	if v := os.Getenv("HTND_BLOOM_FILTER_LEVEL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			bloomBitsPerKey = n
		}
	} else if v := os.Getenv("BLOOM_FILTER_LEVEL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			bloomBitsPerKey = n
		}
	}

	bloomPolicy := bloom.FilterPolicy(bloomBitsPerKey)

	// ────────────────────────────────────────────────
	// MemTable & write stall protection
	// ────────────────────────────────────────────────
	const (
		defaultMemTableMB           = 256
		defaultMemTablesBeforeStall = 64
	)

	memTableBytes := int64(defaultMemTableMB) * 1 << 20
	if v := os.Getenv("HTND_MEMTABLE_SIZE_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			memTableBytes = int64(mb) * 1 << 20
		}
	}

	memTableStopThreshold := defaultMemTablesBeforeStall
	if v := os.Getenv("HTND_MEMTABLE_THRESHOLD"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			memTableStopThreshold = n
		}
	}

	baseFileSize := memTableBytes * int64(memTableStopThreshold)

	// ────────────────────────────────────────────────
	// Block cache size
	// ────────────────────────────────────────────────
	cacheBytes := int64(2024) << 20 // 1 GiB fallback
	if cacheSizeMiB > 0 {
		cacheBytes = int64(cacheSizeMiB) << 20
	}

	if v := os.Getenv("HTND_PEBBLE_CACHE_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			cacheBytes = int64(mb) << 20
		}
	} else if v := os.Getenv("PEBBLE_CACHE_MB"); v != "" {
		if mb, err := strconv.Atoi(v); err == nil && mb > 0 {
			cacheBytes = int64(mb) << 20
		}
	}

	// ────────────────────────────────────────────────
	// Main Pebble options
	// ────────────────────────────────────────────────
	opts := &pebble.Options{
		FormatMajorVersion: pebble.FormatNewest, // modern format, auto-migrates

		Cache: pebble.NewCache(cacheBytes),

		MemTableSize:                uint64(memTableBytes),
		MemTableStopWritesThreshold: memTableStopThreshold,

		// L0 tuning – aggressive for high write throughput
		L0CompactionThreshold:     getEnvInt("HTND_L0_COMPACTION_THRESHOLD", 8),
		L0StopWritesThreshold:     getEnvInt("HTND_L0_STOP_WRITES_THRESHOLD", 48),
		L0CompactionFileThreshold: 1024,

		TargetFileSizes: [7]int64{
			baseFileSize,      // L0
			baseFileSize * 2,  // L1
			baseFileSize * 4,  // L2
			baseFileSize * 8,  // L3
			baseFileSize * 16, // L4
			baseFileSize * 32, // L5
			baseFileSize * 64, // L6
		},

		MaxManifestFileSize: 128 << 20, // 128 MiB
		MaxOpenFiles:        16384,

		// WAL & sync behavior
		DisableWAL:      false,
		WALBytesPerSync: 512 << 10, // 512 KiB
		BytesPerSync:    1 << 20,   // 1 MiB

		// Dynamic compaction workers
		CompactionConcurrencyRange: func() (int, int) { return 2, 4 },

		// Per-level configuration
		Levels: [7]pebble.LevelOptions{
			{ // L0 – fastest ingestion
				BlockSize:      8 << 10,
				IndexBlockSize: 4 << 10,
				Compression:    func() *sstable.CompressionProfile { return sstable.NoCompression },
				FilterPolicy:   bloomPolicy,
			},
			{ // L1
				BlockSize:      8 << 10,
				IndexBlockSize: 4 << 10,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomPolicy,
			},
			{ // L2
				BlockSize:      8 << 10,
				IndexBlockSize: 4 << 10,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomPolicy,
			},
			{ // L3
				BlockSize:      8 << 10,
				IndexBlockSize: 4 << 10,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomPolicy,
			},
			{ // L4
				BlockSize:      8 << 10,
				IndexBlockSize: 4 << 10,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomPolicy,
			},
			{ // L5
				BlockSize:      8 << 10,
				IndexBlockSize: 4 << 10,
				Compression:    func() *sstable.CompressionProfile { return sstable.SnappyCompression },
				FilterPolicy:   bloomPolicy,
			},
			{ // L6 – cold data, better ratio
				BlockSize:      8 << 10,
				IndexBlockSize: 4 << 10,
				Compression:    func() *sstable.CompressionProfile { return sstable.ZstdCompression },
				FilterPolicy:   bloomPolicy,
			},
		},
	}

	// ────────────────────────────────────────────────
	// Optional verbose event logging
	// ────────────────────────────────────────────────
	if envBool("HTND_PEBBLE_LOG_EVENTS") {
		minDurMs := getEnvInt("HTND_PEBBLE_LOG_EVENTS_MIN_MS", 250)
		minDuration := time.Duration(minDurMs) * time.Millisecond

		opts.Logger = pebbleLoggerAdapter{}
		opts.EventListener = newLoggingEventListener(minDuration)
	}

	opts.EnsureDefaults()
	return opts
}

// ──────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────

func getEnvInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}

func envBool(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func newLoggingEventListener(minDuration time.Duration) *pebble.EventListener {
	return &pebble.EventListener{
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
				log.Errorf("[pebble] compaction failed  job=%d  reason=%s  dur=%s  err=%v",
					info.JobID, info.Reason, info.TotalDuration, info.Err)
				return
			}
			if info.TotalDuration >= minDuration {
				log.Infof("[pebble] compaction  job=%d  reason=%s  dur=%s",
					info.JobID, info.Reason, info.TotalDuration)
			}
		},
		FlushEnd: func(info pebble.FlushInfo) {
			if info.Err != nil {
				log.Errorf("[pebble] flush failed  job=%d  reason=%s  dur=%s  err=%v",
					info.JobID, info.Reason, info.TotalDuration, info.Err)
				return
			}
			if info.TotalDuration >= minDuration {
				log.Infof("[pebble] flush  job=%d  reason=%s  input=%d  bytes=%d  ingest=%t  dur=%s",
					info.JobID, info.Reason, info.Input, info.InputBytes, info.Ingest, info.TotalDuration)
			}
		},
		DiskSlow: func(info pebble.DiskSlowInfo) {
			log.Warnf("[pebble] disk slow  op=%s  path=%s  write=%d  dur=%s",
				info.OpType, info.Path, info.WriteSize, info.Duration)
		},
	}
}
