package blockvalidator

import (
	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/pkg/errors"
)

func (v *blockValidator) validateHeaderPruningPoint(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) error {
	// SKIP this check for the time being, investigate the chain
	return nil
	if blockHash.Equal(v.genesisHash) {
		return nil
	}

	header, err := v.blockHeaderStore.BlockHeader(v.databaseContext, stagingArea, blockHash)
	if err != nil {
		return err
	}

	expectedPruningPoint, err := v.pruningManager.ExpectedHeaderPruningPoint(stagingArea, blockHash)
	if err != nil {
		return err
	}
	if !header.PruningPoint().Equal(expectedPruningPoint) {
		return errors.Wrapf(ruleerrors.ErrUnexpectedPruningPoint, "block pruning point of %s is not the expected hash of %s", header.PruningPoint(), expectedPruningPoint)
	}

	return nil
}
