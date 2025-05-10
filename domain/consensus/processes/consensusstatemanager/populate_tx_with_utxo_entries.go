package consensusstatemanager

import (
	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
	"github.com/Hoosat-Oy/HTND/domain/consensus/utils/consensushashing"
)

// PopulateTransactionWithUTXOEntries populates the transaction UTXO entries with data from the virtual's UTXO set.
func (csm *consensusStateManager) PopulateTransactionWithUTXOEntries(
	stagingArea *model.StagingArea, transaction *externalapi.DomainTransaction) error {
	return csm.populateTransactionWithUTXOEntriesFromVirtualOrDiff(stagingArea, transaction, nil)
}

// populateTransactionWithUTXOEntriesFromVirtualOrDiff populates the transaction UTXO entries with data
// from the virtual's UTXO set combined with the provided utxoDiff.
// If utxoDiff == nil UTXO entries are taken from the virtual's UTXO set only
func (csm *consensusStateManager) populateTransactionWithUTXOEntriesFromVirtualOrDiff(stagingArea *model.StagingArea,
	transaction *externalapi.DomainTransaction, utxoDiff externalapi.UTXODiff) error {

	transactionID := consensushashing.TransactionID(transaction)
	log.Tracef("populateTransactionWithUTXOEntriesFromVirtualOrDiff start for transaction %s", transactionID)
	defer log.Tracef("populateTransactionWithUTXOEntriesFromVirtualOrDiff end for transaction %s", transactionID)

	var missingOutpoints []*externalapi.DomainOutpoint
	for i := 0; i < len(transaction.Inputs); i++ {
		// skip all inputs that have a pre-filled utxo entry
		if transaction.Inputs[i].UTXOEntry != nil {
			log.Tracef("Skipping outpoint %s:%d because it is already populated",
				transaction.Inputs[i].PreviousOutpoint.TransactionID, transaction.Inputs[i].PreviousOutpoint.Index)
			continue
		}

		// check if utxoDiff says anything about the input's outpoint
		if utxoDiff != nil {
			if utxoEntry, ok := utxoDiff.ToAdd().Get(&transaction.Inputs[i].PreviousOutpoint); ok {
				log.Tracef("Populating outpoint %s:%d from the given utxoDiff",
					transaction.Inputs[i].PreviousOutpoint.TransactionID, transaction.Inputs[i].PreviousOutpoint.Index)
				transaction.Inputs[i].UTXOEntry = utxoEntry
				continue
			}

			if utxoDiff.ToRemove().Contains(&transaction.Inputs[i].PreviousOutpoint) {
				log.Tracef("Outpoint %s:%d is missing in the given utxoDiff",
					transaction.Inputs[i].PreviousOutpoint.TransactionID, transaction.Inputs[i].PreviousOutpoint.Index)
				missingOutpoints = append(missingOutpoints, &transaction.Inputs[i].PreviousOutpoint)
				continue
			}
		}

		// Check for the input's outpoint in virtual's UTXO set.
		hasUTXOEntry, err := csm.consensusStateStore.HasUTXOByOutpoint(
			csm.databaseContext, stagingArea, &transaction.Inputs[i].PreviousOutpoint)
		if err != nil {
			return err
		}
		if !hasUTXOEntry {
			log.Tracef("Outpoint %s:%d is missing in the database",
				transaction.Inputs[i].PreviousOutpoint.TransactionID, transaction.Inputs[i].PreviousOutpoint.Index)
			missingOutpoints = append(missingOutpoints, &transaction.Inputs[i].PreviousOutpoint)
			continue
		}

		log.Tracef("Populating outpoint %s:%d from the database",
			transaction.Inputs[i].PreviousOutpoint.TransactionID, transaction.Inputs[i].PreviousOutpoint.Index)
		utxoEntry, err := csm.consensusStateStore.UTXOByOutpoint(
			csm.databaseContext, stagingArea, &transaction.Inputs[i].PreviousOutpoint)
		if err != nil {
			return err
		}
		transaction.Inputs[i].UTXOEntry = utxoEntry
	}

	if len(missingOutpoints) > 0 {
		return ruleerrors.NewErrMissingTxOut(missingOutpoints)
	}

	return nil
}

func (csm *consensusStateManager) populateTransactionWithUTXOEntriesFromUTXOSet(
	pruningPoint *externalapi.DomainBlock, iterator externalapi.ReadOnlyUTXOSetIterator) error {

	// Collect the required outpoints from the block
	outpointsForPopulation := make(map[externalapi.DomainOutpoint]interface{})
	for _, transaction := range pruningPoint.Transactions {
		for _, input := range transaction.Inputs {
			outpointsForPopulation[input.PreviousOutpoint] = struct{}{}
		}
	}

	// Collect the UTXO entries from the iterator
	outpointsToUTXOEntries := make(map[externalapi.DomainOutpoint]externalapi.UTXOEntry, len(outpointsForPopulation))
	for ok := iterator.First(); ok; ok = iterator.Next() {
		outpoint, utxoEntry, err := iterator.Get()
		if err != nil {
			return err
		}
		outpointValue := *outpoint
		if _, ok := outpointsForPopulation[outpointValue]; ok {
			outpointsToUTXOEntries[outpointValue] = utxoEntry
		}
		if len(outpointsForPopulation) == len(outpointsToUTXOEntries) {
			break
		}
	}

	// Populate the block with the collected UTXO entries
	var missingOutpoints []*externalapi.DomainOutpoint
	for _, transaction := range pruningPoint.Transactions {
		for _, input := range transaction.Inputs {
			utxoEntry, ok := outpointsToUTXOEntries[input.PreviousOutpoint]
			if !ok {
				missingOutpoints = append(missingOutpoints, &input.PreviousOutpoint)
				continue
			}
			input.UTXOEntry = utxoEntry
		}
	}

	if len(missingOutpoints) > 0 {
		return ruleerrors.NewErrMissingTxOut(missingOutpoints)
	}
	return nil
}
