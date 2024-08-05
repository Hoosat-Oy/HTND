package prefixmanager

import (
	"github.com/Hoosat-Oy/HTND/infrastructure/logger"
	"github.com/Hoosat-Oy/HTND/util/panics"
)

var log = logger.RegisterSubSystem("PRFX")
var spawn = panics.GoroutineWrapperFunc(log)
