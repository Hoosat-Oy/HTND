package ldb

import "github.com/syndtr/goleveldb/leveldb/opt"

// Options is a function that returns a leveldb
// opt.Options struct for opening a database.
func Options() opt.Options {
	return opt.Options{
		Compression:            opt.NoCompression,
		NoSync:                 true,
		WriteBuffer:            32 * opt.MiB,
		BlockCacheCapacity:     32 * opt.MiB,
		OpenFilesCacheCapacity: 256,
		BlockRestartInterval:   16,
		CompactionTableSize:    4 * opt.MiB,
		CompactionTotalSize:    128 * opt.MiB,
	}
}
