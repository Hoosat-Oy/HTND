package protowire

import (
	"github.com/Hoosat-Oy/HTND/infrastructure/logger"
	"github.com/Hoosat-Oy/HTND/util/panics"
)

var log = logger.RegisterSubSystem("PROTO")
var spawn = panics.GoroutineWrapperFunc(log)
