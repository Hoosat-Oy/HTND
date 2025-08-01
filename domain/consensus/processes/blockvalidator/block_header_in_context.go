package blockvalidator

import (
	"fmt"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/consensushashing"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/constants"
	"github.com/Hoosat-Oy/HTND/infrastructure/logger"
	"github.com/pkg/errors"
)

// ValidateHeaderInContext validates block headers in the context of the current
// consensus state
func (v *blockValidator) ValidateHeaderInContext(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash, isBlockWithTrustedData bool) error {
	onEnd := logger.LogAndMeasureExecutionTime(log, "ValidateHeaderInContext")
	defer onEnd()

	header, err := v.blockHeaderStore.BlockHeader(v.databaseContext, stagingArea, blockHash)
	if err != nil {
		return err
	}

	hasValidatedHeader, err := v.hasValidatedHeader(stagingArea, blockHash)
	if err != nil {
		return err
	}

	if !hasValidatedHeader {
		var logErr error
		log.Debug(logger.NewLogClosure(func() string {
			var ghostdagData *externalapi.BlockGHOSTDAGData
			ghostdagData, logErr = v.ghostdagDataStores[0].Get(v.databaseContext, stagingArea, blockHash, false)
			if err != nil {
				return ""
			}

			return fmt.Sprintf("block %s blue score is %d", blockHash, ghostdagData.BlueScore())
		}))

		if logErr != nil {
			return logErr
		}
	}

	err = v.validateMedianTime(stagingArea, header)
	if err != nil {
		return err
	}

	err = v.checkMergeSizeLimit(stagingArea, blockHash)
	if err != nil {
		return err
	}

	// If needed - calculate reachability data right before calling CheckBoundedMergeDepth,
	// since it's used to find a block's finality point.
	// This might not be required if this block's header has previously been received during
	// headers-first synchronization.
	hasReachabilityData, err := v.reachabilityStore.HasReachabilityData(v.databaseContext, stagingArea, blockHash)
	if err != nil {
		return err
	}
	if !hasReachabilityData {
		err = v.reachabilityManager.AddBlock(stagingArea, blockHash)
		if err != nil {
			return err
		}
	}

	//TODO: Think if there is better way to check for indirect parents than the whole reachability.
	if !isBlockWithTrustedData {
		err = v.checkIndirectParents(stagingArea, header)
		if err != nil {
			return err
		}
	}

	err = v.mergeDepthManager.CheckBoundedMergeDepth(stagingArea, blockHash, header, isBlockWithTrustedData)
	if err != nil {
		return err
	}

	err = v.checkDAAScore(stagingArea, blockHash, header)
	if err != nil {
		return err
	}
	v.updateBlockVersion(header)

	if !isBlockWithTrustedData {
		// TODO: Enable these on block v5 after finding reason for the issues with the blocks
		err = v.checkBlueWork(stagingArea, blockHash, header)
		if err != nil {
			return err
		}

		err = v.checkHeaderBlueScore(stagingArea, blockHash, header)
		if err != nil {
			return err
		}

		err = v.validateHeaderPruningPoint(stagingArea, blockHash)
		if err != nil {
			return err
		}
	}

	return nil
}

func (v *blockValidator) updateBlockVersion(header externalapi.BlockHeader) {
	var version uint16 = 1
	daaScore := header.DAAScore()
	if daaScore <= 0 {
		return
	}
	if len(v.POWScores) <= 0 {
		return
	}
	for _, powScore := range v.POWScores {
		if daaScore >= powScore {
			version = version + 1
		}
	}
	constants.BlockVersion = version
}

func (v *blockValidator) hasValidatedHeader(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) (bool, error) {
	exists, err := v.blockStatusStore.Exists(v.databaseContext, stagingArea, blockHash)
	if err != nil {
		return false, err
	}

	if !exists {
		return false, nil
	}

	status, err := v.blockStatusStore.Get(v.databaseContext, stagingArea, blockHash)
	if err != nil {
		return false, err
	}

	return status == externalapi.StatusHeaderOnly, nil
}

// checkParentsIncest validates that no parent is an ancestor of another parent
func (v *blockValidator) checkParentsIncest(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash) error {
	parents, err := v.dagTopologyManagers[0].Parents(stagingArea, blockHash)
	if err != nil {
		return err
	}

	for _, parentA := range parents {
		for _, parentB := range parents {
			if parentA.Equal(parentB) {
				continue
			}

			isAAncestorOfB, err := v.dagTopologyManagers[0].IsAncestorOf(stagingArea, parentA, parentB)
			if err != nil {
				return err
			}

			if isAAncestorOfB {
				return errors.Wrapf(ruleerrors.ErrInvalidParentsRelation, "parent %s is an "+
					"ancestor of another parent %s",
					parentA,
					parentB,
				)
			}
		}
	}
	return nil
}

func (v *blockValidator) validateMedianTime(stagingArea *model.StagingArea, header externalapi.BlockHeader) error {
	if len(header.DirectParents()) == 0 {
		return nil
	}

	// Ensure the timestamp for the block header is not before the
	// median time of the last several blocks (medianTimeBlocks).
	hash := consensushashing.HeaderHash(header)
	pastMedianTime, err := v.pastMedianTimeManager.PastMedianTime(stagingArea, hash)
	if err != nil {
		return err
	}

	if header.TimeInMilliseconds() <= pastMedianTime {
		return errors.Wrapf(ruleerrors.ErrTimeTooOld, "block timestamp of %d is not after expected %d",
			header.TimeInMilliseconds(), pastMedianTime)
	}

	return nil
}

func (v *blockValidator) checkMergeSizeLimit(stagingArea *model.StagingArea, hash *externalapi.DomainHash) error {
	ghostdagData, err := v.ghostdagDataStores[0].Get(v.databaseContext, stagingArea, hash, false)
	if err != nil {
		return err
	}

	mergeSetSize := len(ghostdagData.MergeSetBlues()) + len(ghostdagData.MergeSetReds())

	if uint64(mergeSetSize) > v.mergeSetSizeLimit {
		return errors.Wrapf(ruleerrors.ErrViolatingMergeLimit,
			"The block merges %d blocks > %d merge set size limit", mergeSetSize, v.mergeSetSizeLimit)
	}

	return nil
}

func (v *blockValidator) checkIndirectParents(stagingArea *model.StagingArea, header externalapi.BlockHeader) error {
	expectedParents, err := v.blockParentBuilder.BuildParents(stagingArea, header.DAAScore(), header.DirectParents())
	if err != nil {
		return err
	}

	areParentsEqual := externalapi.ParentsEqual(header.Parents(), expectedParents)
	if !areParentsEqual {
		return errors.Wrapf(ruleerrors.ErrUnexpectedParents, "unexpected indirect block parents")
	}
	return nil
}

func (v *blockValidator) checkDAAScore(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash,
	header externalapi.BlockHeader) error {

	expectedDAAScore, err := v.daaBlocksStore.DAAScore(v.databaseContext, stagingArea, blockHash)
	if err != nil {
		return err
	}
	if header.DAAScore() <= 43334181+500000 {
		return nil
	}
	if header.DAAScore() != expectedDAAScore {
		return errors.Wrapf(ruleerrors.ErrUnexpectedDAAScore, "block DAA score of %d is not the expected value of %d", header.DAAScore(), expectedDAAScore)
	}
	return nil
}

func (v *blockValidator) checkBlueWork(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash,
	header externalapi.BlockHeader) error {

	ghostdagData, err := v.ghostdagDataStores[0].Get(v.databaseContext, stagingArea, blockHash, false)
	if err != nil {
		return err
	}

	expectedBlueWork := ghostdagData.BlueWork()
	headerBlueWork := header.BlueWork()

	if headerBlueWork.Cmp(expectedBlueWork) > 0 {
		return errors.Wrapf(ruleerrors.ErrUnexpectedBlueWork,
			"block blue work %d is ahead of the expected blue work of %d",
			headerBlueWork, expectedBlueWork)
	}
	return nil
}

func (v *blockValidator) checkHeaderBlueScore(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash,
	header externalapi.BlockHeader) error {
	ghostdagData, err := v.ghostdagDataStores[0].Get(v.databaseContext, stagingArea, blockHash, false)
	if err != nil {
		return err
	}

	expectedBlueScore := ghostdagData.BlueScore()
	headerBlueScore := header.BlueScore()

	if headerBlueScore > expectedBlueScore {
		return errors.Wrapf(ruleerrors.ErrUnexpectedBlueScore,
			"block blue score of %d is ahead of the expected blue score of %d",
			headerBlueScore, expectedBlueScore)
	}
	return nil
}
