package ldb

import "github.com/syndtr/goleveldb/leveldb/opt"

// Options is a function that returns a leveldb
// opt.Options struct for opening a database.
func Options() opt.Options {
	return opt.Options{
		Compression:            opt.NoCompression,
		NoSync:                 true,
		WriteBuffer:            128 * opt.MiB,
		BlockCacheCapacity:     128 * opt.MiB,
		OpenFilesCacheCapacity: 1000,
		BlockRestartInterval:   32,
		CompactionTableSize:    8 * opt.MiB,
		CompactionTotalSize:    512 * opt.MiB,
	}
}
