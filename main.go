// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/Hoosat-Oy/HTND/app"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU()) // Set the maximum number of CPUs that can be executing simultaneously
	debug.SetGCPercent(200)              // Set GC to run less aggressively
	debug.SetMemoryLimit(16_000_000_000) // Set memory limit to 16GB
	runtime.SetBlockProfileRate(1)       // Set block profile rate to 1 to enable block profiling
	runtime.SetMutexProfileFraction(1)   // Set mutex profile fraction to 1 to enable mutex profiling

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	if err := app.StartApp(); err != nil {
		os.Exit(1)
	}
}
