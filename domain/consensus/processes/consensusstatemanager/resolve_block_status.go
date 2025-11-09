package consensusstatemanager

import (
	"fmt"

	"github.com/Hoosat-Oy/HTND/util/staging"

	"github.com/Hoosat-Oy/HTND/domain/consensus/database"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/Hoosat-Oy/HTND/infrastructure/logger"
	"github.com/pkg/errors"
)

func (csm *consensusStateManager) resolveBlockStatus(stagingArea *model.StagingArea, blockHash *externalapi.DomainHash,
	useSeparateStagingAreaPerBlock bool) (externalapi.BlockStatus, *model.UTXODiffReversalData, error) {

	onEnd := logger.LogAndMeasureExecutionTime(log, fmt.Sprintf("resolveBlockStatus for %s", blockHash))
	defer onEnd()

	log.Debugf("Getting a list of all blocks in the selected "+
		"parent chain of %s that have no yet resolved their status", blockHash)
	unverifiedBlocks, err := csm.getUnverifiedChainBlocks(stagingArea, blockHash)
	if err != nil {
		return 0, nil, err
	}
	log.Debugf("Got %d unverified blocks in the selected parent "+
		"chain of %s: %s", len(unverifiedBlocks), blockHash, unverifiedBlocks)

	// If there's no unverified blocks in the given block's chain - this means the given block already has a
	// UTXO-verified status, and therefore it should be retrieved from the store and returned
	if len(unverifiedBlocks) == 0 {
		log.Debugf("There are not unverified blocks in %s's selected parent chain. "+
			"This means that the block already has a UTXO-verified status.", blockHash)
		status, err := csm.blockStatusStore.Get(csm.databaseContext, stagingArea, blockHash)
		if err != nil {
			return 0, nil, err
		}
		log.Debugf("Block %s's status resolved to: %s", blockHash, status)
		return status, nil, nil
	}

	log.Debugf("Finding the status of the selected parent of %s", blockHash)
	selectedParentHash, selectedParentStatus, selectedParentUTXOSet, err := csm.selectedParentInfo(stagingArea, unverifiedBlocks)
	if err != nil {
		return 0, nil, err
	}
	log.Debugf("The status of the selected parent of %s is: %s", blockHash, selectedParentStatus)

	log.Debugf("Resolving the unverified blocks' status in reverse order (past to present)")
	var blockStatus externalapi.BlockStatus

	previousBlockHash := selectedParentHash
	previousBlockUTXOSet := selectedParentUTXOSet
	var oneBeforeLastResolvedBlockUTXOSet externalapi.UTXODiff
	var oneBeforeLastResolvedBlockHash *externalapi.DomainHash

	for i := len(unverifiedBlocks) - 1; i >= 0; i-- {
		unverifiedBlockHash := unverifiedBlocks[i]

		stagingAreaForCurrentBlock := stagingArea
		isResolveTip := i == 0
		useSeparateStagingArea := useSeparateStagingAreaPerBlock && !isResolveTip
		if useSeparateStagingArea {
			stagingAreaForCurrentBlock = model.NewStagingArea()
		}

		if selectedParentStatus == externalapi.StatusDisqualifiedFromChain {
			// Even when parent is disqualified, we need to calculate and stage UTXO diff for the child
			// to maintain a complete diff chain for UTXO restoration
			blockStatus = externalapi.StatusDisqualifiedFromChain

			// Calculate the block's UTXO state even though it will be disqualified
			blockGHOSTDAGData, err := csm.ghostdagDataStore.Get(csm.databaseContext, stagingAreaForCurrentBlock, unverifiedBlockHash, false)
			if err != nil {
				return 0, nil, err
			}

			pastUTXOSet, acceptanceData, multiset, err := csm.calculatePastUTXOAndAcceptanceDataWithSelectedParentUTXO(
				stagingAreaForCurrentBlock, unverifiedBlockHash, previousBlockUTXOSet, blockGHOSTDAGData)
			if err != nil {
				return 0, nil, err
			}

			// Stage the calculated data for consistency
			csm.acceptanceDataStore.Stage(stagingAreaForCurrentBlock, unverifiedBlockHash, acceptanceData)
			csm.multisetStore.Stage(stagingAreaForCurrentBlock, unverifiedBlockHash, multiset)

			// Stage the UTXO diff with selectedParent as diffChild
			utxoDiff, err := previousBlockUTXOSet.DiffFrom(pastUTXOSet)
			if err != nil {
				return 0, nil, err
			}
			csm.stageDiff(stagingAreaForCurrentBlock, unverifiedBlockHash, utxoDiff, previousBlockHash)

			previousBlockUTXOSet = pastUTXOSet
		} else {
			oneBeforeLastResolvedBlockUTXOSet = previousBlockUTXOSet
			oneBeforeLastResolvedBlockHash = previousBlockHash

			blockStatus, previousBlockUTXOSet, err = csm.resolveSingleBlockStatus(
				stagingAreaForCurrentBlock, unverifiedBlockHash, previousBlockHash, previousBlockUTXOSet, isResolveTip)
			if err != nil {
				return 0, nil, err
			}
		}

		csm.blockStatusStore.Stage(stagingAreaForCurrentBlock, unverifiedBlockHash, blockStatus)
		selectedParentStatus = blockStatus
		log.Debugf("Block %s status resolved to `%s`, finished %d/%d of unverified blocks",
			unverifiedBlockHash, blockStatus, len(unverifiedBlocks)-i, len(unverifiedBlocks))

		if useSeparateStagingArea {
			err := staging.CommitAllChanges(csm.databaseContext, stagingAreaForCurrentBlock)
			if err != nil {
				return 0, nil, err
			}
		}
		previousBlockHash = unverifiedBlockHash
	}

	var reversalData *model.UTXODiffReversalData
	if blockStatus == externalapi.StatusUTXOValid && len(unverifiedBlocks) > 1 {
		log.Debugf("Preparing data for reversing the UTXODiff")
		// During resolveSingleBlockStatus, all unverifiedBlocks (excluding the tip) were assigned their selectedParent
		// as their UTXODiffChild.
		// Now that the whole chain has been resolved - we can reverse the UTXODiffs, to create shorter UTXODiffChild paths.
		// However, we can't do this right now, because the tip of the chain is not yet committed, so we prepare the
		// needed data (tip's selectedParent and selectedParent's UTXODiff)
		selectedParentUTXODiff, err := previousBlockUTXOSet.DiffFrom(oneBeforeLastResolvedBlockUTXOSet)
		if err != nil {
			return 0, nil, err
		}

		reversalData = &model.UTXODiffReversalData{
			SelectedParentHash:     oneBeforeLastResolvedBlockHash,
			SelectedParentUTXODiff: selectedParentUTXODiff,
		}
	}

	return blockStatus, reversalData, nil
}

// selectedParentInfo returns the hash and status of the selectedParent of the last block in the unverifiedBlocks
// chain, in addition, if the status is UTXOValid, it return it's pastUTXOSet
func (csm *consensusStateManager) selectedParentInfo(
	stagingArea *model.StagingArea, unverifiedBlocks []*externalapi.DomainHash) (
	*externalapi.DomainHash, externalapi.BlockStatus, externalapi.UTXODiff, error) {

	log.Tracef("findSelectedParentStatus start")
	defer log.Tracef("findSelectedParentStatus end")

	lastUnverifiedBlock := unverifiedBlocks[len(unverifiedBlocks)-1]
	if lastUnverifiedBlock.Equal(csm.genesisHash) {
		log.Debugf("the most recent unverified block is the genesis block, "+
			"which by definition has status: %s", externalapi.StatusUTXOValid)
		utxoDiff, err := csm.utxoDiffStore.UTXODiff(csm.databaseContext, stagingArea, lastUnverifiedBlock)
		if err != nil {
			return nil, 0, nil, err
		}
		return lastUnverifiedBlock, externalapi.StatusUTXOValid, utxoDiff, nil
	}
	lastUnverifiedBlockGHOSTDAGData, err := csm.ghostdagDataStore.Get(csm.databaseContext, stagingArea, lastUnverifiedBlock, false)
	if database.IsNotFoundError(err) {
		log.Infof("selectedParentInfo failed to retrieve with %s\n", lastUnverifiedBlock)
		return nil, 0, nil, err
	}
	if err != nil {
		return nil, 0, nil, err
	}
	selectedParent := lastUnverifiedBlockGHOSTDAGData.SelectedParent()
	selectedParentStatus, err := csm.blockStatusStore.Get(csm.databaseContext, stagingArea, selectedParent)
	if database.IsNotFoundError(err) {
		log.Infof("selectedParentInfo failed to retrieve with %s\n", selectedParent)
		return nil, 0, nil, err
	}
	if err != nil {
		return nil, 0, nil, err
	}
	if selectedParentStatus != externalapi.StatusUTXOValid {
		return selectedParent, selectedParentStatus, nil, nil
	}

	selectedParentUTXOSet, err := csm.restorePastUTXO(stagingArea, selectedParent)
	if err != nil {
		return nil, 0, nil, err
	}
	return selectedParent, selectedParentStatus, selectedParentUTXOSet, nil
}

func (csm *consensusStateManager) getUnverifiedChainBlocks(stagingArea *model.StagingArea,
	blockHash *externalapi.DomainHash) ([]*externalapi.DomainHash, error) {

	log.Tracef("getUnverifiedChainBlocks start for block %s", blockHash)
	defer log.Tracef("getUnverifiedChainBlocks end for block %s", blockHash)

	var unverifiedBlocks []*externalapi.DomainHash
	currentHash := blockHash
	for {
		log.Tracef("Getting status for block %s", currentHash)
		currentBlockStatus, err := csm.blockStatusStore.Get(csm.databaseContext, stagingArea, currentHash)
		if database.IsNotFoundError(err) {
			log.Infof("getUnverifiedChainBlocks failed to retrieve with %s\n", currentHash)
			return nil, err
		}
		if err != nil {
			return nil, err
		}
		if currentBlockStatus != externalapi.StatusUTXOPendingVerification {
			log.Tracef("Block %s has status %s. Returning all the "+
				"unverified blocks prior to it: %s", currentHash, currentBlockStatus, unverifiedBlocks)
			return unverifiedBlocks, nil
		}

		log.Tracef("Block %s is unverified. Adding it to the unverified block collection", currentHash)
		unverifiedBlocks = append(unverifiedBlocks, currentHash)

		currentBlockGHOSTDAGData, err := csm.ghostdagDataStore.Get(csm.databaseContext, stagingArea, currentHash, false)
		if database.IsNotFoundError(err) {
			log.Infof("getUnverifiedChainBlocks failed to retrieve with %s\n", currentHash)
			return nil, err
		}
		if err != nil {
			return nil, err
		}

		if currentBlockGHOSTDAGData.SelectedParent() == nil {
			log.Tracef("Genesis block reached. Returning all the "+
				"unverified blocks prior to it: %s", unverifiedBlocks)
			return unverifiedBlocks, nil
		}

		currentHash = currentBlockGHOSTDAGData.SelectedParent()
	}
}

func (csm *consensusStateManager) resolveSingleBlockStatus(stagingArea *model.StagingArea,
	blockHash, selectedParentHash *externalapi.DomainHash, selectedParentPastUTXOSet externalapi.UTXODiff, isResolveTip bool) (
	externalapi.BlockStatus, externalapi.UTXODiff, error) {

	onEnd := logger.LogAndMeasureExecutionTime(log, fmt.Sprintf("resolveSingleBlockStatus for %s", blockHash))
	defer onEnd()

	log.Tracef("Calculating pastUTXO and acceptance data and multiset for block %s", blockHash)
	blockGHOSTDAGData, err := csm.ghostdagDataStore.Get(csm.databaseContext, stagingArea, blockHash, false)
	if database.IsNotFoundError(err) {
		log.Infof("resolveSingleBlockStatus failed to retrieve with %s\n", blockHash)
		return 0, nil, err
	}
	if err != nil {
		return 0, nil, err
	}

	// Ensure all blocks in the merge set have their acceptance data calculated
	// This is necessary because acceptance data calculation depends on merge set blocks
	// err = csm.ensureMergeSetAcceptanceData(stagingArea, blockHash, blockGHOSTDAGData)
	// if err != nil {
	// 	return 0, nil, err
	// }

	pastUTXOSet, acceptanceData, multiset, err := csm.calculatePastUTXOAndAcceptanceDataWithSelectedParentUTXO(
		stagingArea, blockHash, selectedParentPastUTXOSet, blockGHOSTDAGData)
	if err != nil {
		return 0, nil, err
	}

	block, err := csm.blockStore.Block(csm.databaseContext, stagingArea, blockHash)
	if err != nil {
		return 0, nil, err
	}

	log.Tracef("verifying the UTXO of block %s", blockHash)
	err = csm.verifyUTXO(stagingArea, block, blockHash, pastUTXOSet, acceptanceData, multiset)
	isDisqualified := false
	if err != nil {
		if errors.As(err, &ruleerrors.RuleError{}) {
			log.Debugf("UTXO verification for block %s failed: %s", blockHash, err)
			isDisqualified = true
		} else {
			return 0, nil, err
		}
	} else {
		log.Debugf("UTXO verification for block %s passed", blockHash)
	}

	// Stage the acceptance data and multiset even for disqualified blocks to maintain data consistency
	log.Tracef("Staging the calculated acceptance data of block %s", blockHash)
	csm.acceptanceDataStore.Stage(stagingArea, blockHash, acceptanceData)

	log.Tracef("Staging the multiset of block %s", blockHash)
	csm.multisetStore.Stage(stagingArea, blockHash, multiset)

	if csm.genesisHash.Equal(blockHash) {
		log.Tracef("Staging the utxoDiff of genesis")
		csm.stageDiff(stagingArea, blockHash, pastUTXOSet, nil)
		if isDisqualified {
			return externalapi.StatusDisqualifiedFromChain, nil, nil
		}
		return externalapi.StatusUTXOValid, nil, nil
	}

	oldSelectedTip, err := csm.virtualSelectedParent(stagingArea)
	if err != nil {
		return 0, nil, err
	}

	// Stage UTXO diff for all blocks (including disqualified ones) to maintain a complete diff chain
	if isResolveTip {
		oldSelectedTipUTXOSet, err := csm.restorePastUTXO(stagingArea, oldSelectedTip)
		if err != nil {
			return 0, nil, err
		}
		isNewSelectedTip, err := csm.isNewSelectedTip(stagingArea, blockHash, oldSelectedTip)
		if err != nil {
			return 0, nil, err
		}

		if isNewSelectedTip {
			log.Debugf("Block %s is the new selected tip, therefore setting it as old selected tip's diffChild", blockHash)

			updatedOldSelectedTipUTXOSet, err := pastUTXOSet.DiffFrom(oldSelectedTipUTXOSet)
			if err != nil {
				return 0, nil, err
			}
			log.Debugf("Setting the old selected tip's (%s) diffChild to be the new selected tip (%s)",
				oldSelectedTip, blockHash)
			csm.stageDiff(stagingArea, oldSelectedTip, updatedOldSelectedTipUTXOSet, blockHash)

			log.Tracef("Staging the utxoDiff of block %s, with virtual as diffChild", blockHash)
			csm.stageDiff(stagingArea, blockHash, pastUTXOSet, nil)
		} else {
			log.Debugf("Block %s is the tip of currently resolved chain, but not the new selected tip,"+
				"therefore setting it's utxoDiffChild to be the current selectedTip %s", blockHash, oldSelectedTip)
			utxoDiff, err := oldSelectedTipUTXOSet.DiffFrom(pastUTXOSet)
			if err != nil {
				return 0, nil, err
			}
			csm.stageDiff(stagingArea, blockHash, utxoDiff, oldSelectedTip)
		}
	} else {
		// If the block is not the tip of the currently resolved chain, we set it's diffChild to be the selectedParent,
		// this is a temporary measure to ensure there's a restore path to all blocks at all times.
		// Later down the process, the diff will be reversed in reverseUTXODiffs.
		log.Debugf("Block %s is not the new selected tip, and is not the tip of the currently verified chain, "+
			"therefore temporarily setting selectedParent as it's diffChild", blockHash)
		utxoDiff, err := selectedParentPastUTXOSet.DiffFrom(pastUTXOSet)
		if err != nil {
			return 0, nil, err
		}

		csm.stageDiff(stagingArea, blockHash, utxoDiff, selectedParentHash)
	}

	// Return the appropriate status
	if isDisqualified {
		return externalapi.StatusDisqualifiedFromChain, nil, nil
	}
	return externalapi.StatusUTXOValid, pastUTXOSet, nil
}

func (csm *consensusStateManager) isNewSelectedTip(stagingArea *model.StagingArea,
	blockHash, oldSelectedTip *externalapi.DomainHash) (bool, error) {

	newSelectedTip, err := csm.ghostdagManager.ChooseSelectedParent(stagingArea, blockHash, oldSelectedTip)
	if database.IsNotFoundError(err) {
		log.Infof("isNewSelectedTip failed to retrieve with %s\n", oldSelectedTip)
		return false, err
	}
	if err != nil {
		return false, err
	}

	return blockHash.Equal(newSelectedTip), nil
}

func (csm *consensusStateManager) virtualSelectedParent(stagingArea *model.StagingArea) (*externalapi.DomainHash, error) {
	virtualGHOSTDAGData, err := csm.ghostdagDataStore.Get(csm.databaseContext, stagingArea, model.VirtualBlockHash, false)
	if err != nil {
		return nil, err
	}

	return virtualGHOSTDAGData.SelectedParent(), nil
}

// ensureMergeSetAcceptanceData ensures that all blocks in the merge set have their acceptance data calculated and staged.
// This is necessary because when calculating acceptance data for a block, it requires acceptance data from all blocks
// in its merge set. During IBD, blocks may be added without having their acceptance data calculated, so we need to
// calculate it on-demand when it's needed.
func (csm *consensusStateManager) ensureMergeSetAcceptanceData(stagingArea *model.StagingArea,
	blockHash *externalapi.DomainHash, blockGHOSTDAGData *externalapi.BlockGHOSTDAGData) error {

	log.Tracef("ensureMergeSetAcceptanceData start for block %s", blockHash)
	defer log.Tracef("ensureMergeSetAcceptanceData end for block %s", blockHash)

	// Get all blocks in the merge set
	mergeSetHashes, err := csm.ghostdagManager.GetSortedMergeSet(stagingArea, blockHash)
	if err != nil {
		return err
	}

	// For each block in the merge set, check if it has acceptance data
	for _, mergeSetBlockHash := range mergeSetHashes {
		// First, get the block to check if it's header-only or if it exists
		// This is more efficient than checking acceptance data store first during IBD
		mergeSetBlock, err := csm.blockStore.Block(csm.databaseContext, stagingArea, mergeSetBlockHash)
		if database.IsNotFoundError(err) {
			// Block doesn't exist yet (not downloaded during IBD), skip it
			log.Tracef("Merge set block %s not found in database, skipping acceptance data calculation", mergeSetBlockHash)
			continue
		}
		if err != nil {
			return err
		}

		// Skip header-only blocks as they cannot have acceptance data calculated
		// This check MUST come before the acceptance data store lookup to avoid excessive DB queries during IBD
		isHeaderOnlyBlock := len(mergeSetBlock.Transactions) == 0
		if isHeaderOnlyBlock {
			log.Tracef("Merge set block %s is header-only, skipping acceptance data calculation", mergeSetBlockHash)
			continue
		}

		// Check if acceptance data already exists
		_, err = csm.acceptanceDataStore.Get(csm.databaseContext, stagingArea, mergeSetBlockHash)
		if err == nil {
			// Acceptance data exists, skip this block
			continue
		}
		if !database.IsNotFoundError(err) {
			// Real error, not just missing data
			return err
		}

		// Acceptance data is missing, we need to calculate it
		log.Debugf("Acceptance data missing for merge set block %s, calculating now", mergeSetBlockHash)

		// Calculate and stage acceptance data for this block
		// We need to recursively ensure its dependencies are met first
		mergeSetBlockGHOSTDAGData, err := csm.ghostdagDataStore.Get(csm.databaseContext, stagingArea, mergeSetBlockHash, false)
		if err != nil {
			return err
		}

		// Recursively ensure this block's merge set has acceptance data
		err = csm.ensureMergeSetAcceptanceData(stagingArea, mergeSetBlockHash, mergeSetBlockGHOSTDAGData)
		if err != nil {
			// If we can't ensure merge set acceptance data (e.g., missing blocks during IBD),
			// skip this block and continue. It will be resolved later when all dependencies are available.
			log.Debugf("Cannot ensure merge set acceptance data for %s, skipping: %s", mergeSetBlockHash, err)
			continue
		}

		// Now calculate acceptance data for this block
		_, acceptanceData, multiset, err := csm.CalculatePastUTXOAndAcceptanceData(stagingArea, mergeSetBlockHash)
		if err != nil {
			// If we can't calculate acceptance data (e.g., missing blocks in the merge set during IBD),
			// skip this block. It will be resolved later when all dependencies are available.
			log.Debugf("Cannot calculate acceptance data for merge set block %s, skipping: %s", mergeSetBlockHash, err)
			continue
		}

		// Stage the acceptance data and multiset
		log.Debugf("Staging acceptance data and multiset for merge set block %s", mergeSetBlockHash)
		csm.acceptanceDataStore.Stage(stagingArea, mergeSetBlockHash, acceptanceData)
		csm.multisetStore.Stage(stagingArea, mergeSetBlockHash, multiset)
	}

	return nil
}
