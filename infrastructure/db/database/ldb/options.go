package ldb

import "github.com/syndtr/goleveldb/leveldb/opt"

// Options returns a leveldb opt.Options struct optimized for Kaspa's high block rate.
func Options() opt.Options {
	return opt.Options{
		Compression:            opt.NoCompression, // No compression for minimal CPU overhead
		NoSync:                 true,              // Disable sync for maximum write throughput
		WriteBuffer:            96 * opt.MiB,      // Slightly reduced to allocate more memory to read cache
		BlockCacheCapacity:     512 * opt.MiB,     // Significantly increased for better read performance
		OpenFilesCacheCapacity: 2048,              // Higher to support many SST files for read-heavy workload
		BlockRestartInterval:   8,                 // Reduced for smaller block sizes, faster read scans
		CompactionTableSize:    32 * opt.MiB,      // Larger tables to reduce file count and read amplification
		CompactionTotalSize:    1024 * opt.MiB,    // Larger to delay compactions, reducing read/write interference
	}
}
