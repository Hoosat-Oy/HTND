package main

import (
	"github.com/Hoosat-Oy/HTND/infrastructure/logger"
)

var (
	backendLog = logger.NewBackend()
	log        = backendLog.Logger("RPIC")
)
