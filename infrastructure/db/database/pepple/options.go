package pepple

import (
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

// Options returns a pebble.Options struct optimized for Kaspa's high block rate (33 blocks/s, 10,000 tx/block).
func Options() *pebble.Options {
	// Define a Bloom filter with 8 bits per key for space efficiency
	bloomFilter := bloom.FilterPolicy(8)

	opts := &pebble.Options{
		// Sync settings: Balance durability and performance for 100 ms block time
		BytesPerSync:       2 * 1024 * 1024,                                       // 2 MB to reduce sync frequency
		WALBytesPerSync:    2 * 1024 * 1024,                                       // Sync WAL less frequently
		WALMinSyncInterval: func() time.Duration { return 10 * time.Millisecond }, // Async WAL writes within block time
		DisableWAL:         false,                                                 // Ensure durability for blockchain data
		FlushSplitBytes:    1 * 1024 * 1024,                                       // 1 MB for WAL file splitting

		// Memory settings: Handle write throughput (~30.15 MB/s)
		MemTableSize:                64 * 1024 * 1024,        // 64 MB
		MemTableStopWritesThreshold: 4,                       // Allow up to 4 MemTables (256 MB total)
		MaxConcurrentCompactions:    func() int { return 4 }, // Parallel compactions

		// File settings: Support high concurrency and large SSTables
		MaxOpenFiles:        1000,              // Sufficient for SSTables at lower write rate
		MaxManifestFileSize: 128 * 1024 * 1024, // Sufficient for high throughput

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
