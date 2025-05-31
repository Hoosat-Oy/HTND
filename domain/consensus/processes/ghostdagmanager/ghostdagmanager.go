package ghostdagmanager

import (
	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/lrucache"
)

// ghostdagManager resolves and manages GHOSTDAG block data
type ghostdagManager struct {
	blueAnticoneSizeCache *lrucache.LRUCache
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
	return &ghostdagManager{
		blueAnticoneSizeCache: lrucache.New(9000, true),
		databaseContext:       databaseContext,
		dagTopologyManager:    dagTopologyManager,
		ghostdagDataStore:     ghostdagDataStore,
		headerStore:           headerStore,
		k:                     k,
		genesisHash:           genesisHash,
	}
}
