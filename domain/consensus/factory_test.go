package consensus

import (
	"io/ioutil"
	"testing"

	"github.com/Hoosat-Oy/HTND/domain/prefixmanager/prefix"
	"github.com/Hoosat-Oy/HTND/infrastructure/db/database/pepple"

	"github.com/Hoosat-Oy/HTND/domain/dagconfig"
)

func TestNewConsensus(t *testing.T) {
	f := NewFactory()

	config := &Config{Params: dagconfig.DevnetParams}

	tmpDir, err := ioutil.TempDir("", "TestNewConsensus")
	if err != nil {
		return
	}

	db, err := pepple.NewPeppleDB(tmpDir, 8)
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
