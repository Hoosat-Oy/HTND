package ldb

import (
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

// Options returns a leveldb opt.Options struct optimized for Kaspa's high block rate.
func Options() opt.Options {
	return opt.Options{
		Compression: opt.SnappyCompression,     // Good for reducing I/O
		NoSync:      true,                      // Boosts write throughput, but risks data loss
		Filter:      filter.NewBloomFilter(10), // Bloom filter for read efficiency
	}
}
