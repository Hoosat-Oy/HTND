// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/Hoosat-Oy/HTND/app"
)

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	if err := app.StartApp(); err != nil {
		os.Exit(1)
	}
}
