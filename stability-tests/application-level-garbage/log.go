package main

import (
	"github.com/Hoosat-Oy/HTND/infrastructure/logger"
	"github.com/Hoosat-Oy/HTND/util/panics"
)

var (
	backendLog = logger.NewBackend()
	log        = backendLog.Logger("APLG")
	spawn      = panics.GoroutineWrapperFunc(log)
)
