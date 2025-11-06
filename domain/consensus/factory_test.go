package consensus

import (
	"os"
	"testing"

	"github.com/Hoosat-Oy/HTND/domain/prefixmanager/prefix"
	"github.com/Hoosat-Oy/HTND/infrastructure/db/database/pebble"

	"github.com/Hoosat-Oy/HTND/domain/dagconfig"
)

func TestNewConsensus(t *testing.T) {
	f := NewFactory()

	config := &Config{Params: dagconfig.DevnetParams}

	tmpDir, err := os.MkdirTemp("", "TestNewConsensus")
	if err != nil {
		return
	}

	db, err := pebble.NewPebbleDB(tmpDir, 8)
	if err != nil {
		t.Fatalf("error in NewLevelDB: %s", err)
	}

	_, shouldMigrate, err := f.NewConsensus(config, db, &prefix.Prefix{}, nil)
	if err != nil {
		t.Fatalf("error in NewConsensus: %+v", err)
	}

	if shouldMigrate {
		t.Fatalf("A fresh consensus should never return shouldMigrate=true")
	}
}
