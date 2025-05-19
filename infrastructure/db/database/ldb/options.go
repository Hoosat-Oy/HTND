package ldb

import "github.com/syndtr/goleveldb/leveldb/opt"

// Options is a function that returns a leveldb
// opt.Options struct for opening a database.
func Options() opt.Options {
	return opt.Options{
		Compression:            opt.NoCompression, // Skip compression for speed
		NoSync:                 true,              // Skip fsync for faster writes (not durable)
		WriteBuffer:            64 * opt.MiB,      // Larger memtable
		BlockCacheCapacity:     64 * opt.MiB,      // Larger block cache for reads
		OpenFilesCacheCapacity: 500,               // Avoid file open/close overhead
		BlockRestartInterval:   32,                // Slightly larger blocks
	}
}
