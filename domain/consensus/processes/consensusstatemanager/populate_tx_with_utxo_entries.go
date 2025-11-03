package consensusstatemanager

import (
	"runtime"
	"sync"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model"
	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
	"github.com/Hoosat-Oy/HTND/domain/consensus/ruleerrors"
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

	var (
		missingOutpoints []*externalapi.DomainOutpoint
		missingMu        sync.Mutex // Protects missingOutpoints
		outpointsMu      sync.Mutex // Protects transaction.Inputs
		errMu            sync.Mutex // Protects error collection
		firstErr         error      // Stores the first error encountered
		wg               sync.WaitGroup
	)

	// Use a bounded worker pool to process inputs concurrently without overwhelming the DB.
	parallelism := runtime.NumCPU()
	if parallelism < 1 {
		parallelism = 1
	}
	sem := make(chan struct{}, parallelism)

	// Process each input in a separate goroutine (bounded by sem)
	for i := 0; i < len(transaction.Inputs); i++ {
		wg.Add(1)
		go func(index int) {
			// Acquire a worker slot
			sem <- struct{}{}
			defer wg.Done()
			// Release the worker slot
			defer func() { <-sem }()

			// Skip if error already occurred
			errMu.Lock()
			if firstErr != nil {
				errMu.Unlock()
				return
			}
			errMu.Unlock()

			// Skip inputs with pre-filled UTXO entries
			if transaction.Inputs[index].UTXOEntry != nil {
				return
			}

			// Check utxoDiff if provided
			if utxoDiff != nil {
				if utxoEntry, ok := utxoDiff.ToAdd().Get(&transaction.Inputs[index].PreviousOutpoint); ok {
					transaction.Inputs[index].UTXOEntry = utxoEntry
					return
				}

				if utxoDiff.ToRemove().Contains(&transaction.Inputs[index].PreviousOutpoint) {
					missingMu.Lock()
					missingOutpoints = append(missingOutpoints, &transaction.Inputs[index].PreviousOutpoint)
					missingMu.Unlock()
					return
				}
			}

			// Check virtual's UTXO set
			outpointsMu.Lock()
			utxoEntry, hasUTXOEntry, err := csm.consensusStateStore.UTXOByOutpoint(csm.databaseContext, stagingArea, &transaction.Inputs[index].PreviousOutpoint)
			outpointsMu.Unlock()
			if !hasUTXOEntry {
				missingMu.Lock()
				missingOutpoints = append(missingOutpoints, &transaction.Inputs[index].PreviousOutpoint)
				missingMu.Unlock()
				return
			}
			if err != nil {
				errMu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				errMu.Unlock()
				return
			}

			transaction.Inputs[index].UTXOEntry = utxoEntry
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Check for errors
	if firstErr != nil {
		return firstErr
	}

	// Check for missing outpoints
	if len(missingOutpoints) > 0 {
		return ruleerrors.NewErrMissingTxOut(missingOutpoints)
	}

	return nil
}

func (csm *consensusStateManager) populateTransactionWithUTXOEntriesFromUTXOSet(
	pruningPoint *externalapi.DomainBlock, iterator externalapi.ReadOnlyUTXOSetIterator) error {

	// Build an index from required outpoints to the inputs that need them.
	// This lets us assign UTXO entries directly while scanning the iterator,
	// and avoid a second pass over all transactions.
	outpointsToInputs := make(map[externalapi.DomainOutpoint][]*externalapi.DomainTransactionInput)
	uniqueNeeded := 0
	for _, transaction := range pruningPoint.Transactions {
		for i := range transaction.Inputs {
			in := transaction.Inputs[i]
			// Add pointer to the input so we can fill it directly when found.
			slice := outpointsToInputs[in.PreviousOutpoint]
			if slice == nil {
				uniqueNeeded++
			}
			outpointsToInputs[in.PreviousOutpoint] = append(slice, in)
		}
	}

	// Walk the iterator once and satisfy inputs as we find matching outpoints.
	// Break early as soon as all required outpoints were found.
	found := 0
	for ok := iterator.First(); ok; ok = iterator.Next() {
		outpoint, utxoEntry, err := iterator.Get()
		if err != nil {
			return err
		}
		if inputs, ok := outpointsToInputs[*outpoint]; ok {
			for _, in := range inputs {
				in.UTXOEntry = utxoEntry
			}
			delete(outpointsToInputs, *outpoint)
			found++
			if found == uniqueNeeded {
				break
			}
		}
	}

	// Collect any inputs still missing a UTXO entry.
	if len(outpointsToInputs) > 0 {
		var missingOutpoints []*externalapi.DomainOutpoint
		for _, inputs := range outpointsToInputs {
			for _, in := range inputs {
				// Report per-missing-input, matching previous behavior
				// where each missing input contributed an outpoint.
				missingOutpoints = append(missingOutpoints, &in.PreviousOutpoint)
			}
		}
		return ruleerrors.NewErrMissingTxOut(missingOutpoints)
	}
	return nil
}
