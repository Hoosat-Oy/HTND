package ldb

import "github.com/syndtr/goleveldb/leveldb/opt"

// Options returns a leveldb opt.Options struct optimized for Kaspa's high block rate.
func Options() opt.Options {
	return opt.Options{
		Compression:            opt.SnappyCompression, // Use Snappy to reduce I/O
		NoSync:                 true,                  // Keep for write throughput
		WriteBuffer:            32 * opt.MiB,          // Smaller buffer for smaller SSTs
		BlockCacheCapacity:     512 * opt.MiB,         // Keep for read performance
		OpenFilesCacheCapacity: 512,                   // Reduce slightly to limit open files
		BlockRestartInterval:   8,                     // Keep for faster read scans
		CompactionTableSize:    8 * opt.MiB,           // Smaller tables for faster compactions
		CompactionTotalSize:    1024 * opt.MiB,        // Smaller total size to trigger compactions sooner
	}
}
