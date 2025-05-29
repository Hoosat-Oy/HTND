package ghostdagmanager

import (
	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	lru "github.com/hashicorp/golang-lru"
)

// ghostdagManager resolves and manages GHOSTDAG block data
type ghostdagManager struct {
	blueAnticoneSizeCache *lru.Cache
	databaseContext       model.DBReader
	dagTopologyManager    model.DAGTopologyManager
	ghostdagDataStore     model.GHOSTDAGDataStore
	headerStore           model.BlockHeaderStore

	k           externalapi.KType
	genesisHash *externalapi.DomainHash
}

// New instantiates a new GHOSTDAGManager
func New(
	databaseContext model.DBReader,
	dagTopologyManager model.DAGTopologyManager,
	ghostdagDataStore model.GHOSTDAGDataStore,
	headerStore model.BlockHeaderStore,
	k externalapi.KType,
	genesisHash *externalapi.DomainHash) model.GHOSTDAGManager {
	cache, _ := lru.New(9000)
	return &ghostdagManager{
		blueAnticoneSizeCache: cache,
		databaseContext:       databaseContext,
		dagTopologyManager:    dagTopologyManager,
		ghostdagDataStore:     ghostdagDataStore,
		headerStore:           headerStore,
		k:                     k,
		genesisHash:           genesisHash,
	}
}
