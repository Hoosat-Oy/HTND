package ldb

import "github.com/syndtr/goleveldb/leveldb/opt"

// Options returns a leveldb opt.Options struct optimized for Kaspa's high block rate.
func Options() opt.Options {
	return opt.Options{
		Compression:            opt.NoCompression, // Keep for low CPU overhead
		NoSync:                 true,              // Keep for high write throughput
		WriteBuffer:            64 * opt.MiB,      // Increase to reduce write frequency
		BlockCacheCapacity:     128 * opt.MiB,     // Increase for better read performance
		OpenFilesCacheCapacity: 512,               // Increase to handle more SST files
		BlockRestartInterval:   16,                // Keep default, adjust if needed
		CompactionTableSize:    8 * opt.MiB,       // Larger tables to reduce file count
		CompactionTotalSize:    256 * opt.MiB,     // Larger to delay compactions
	}
}
