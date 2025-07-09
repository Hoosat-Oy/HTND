package pepple

import (
	"github.com/Hoosat-Oy/HTND/infrastructure/db/database"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
)

// PeppleDBTransaction is a thin wrapper around Pebble batches.
// It supports both get and put.
// Note that reads are done from the Database directly, so if another transaction changed the data,
// you will read the new data, and not the one from the time the transaction was opened.
// Note: As it's currently implemented, if one puts data into the transaction
// then it will not be available to get within the same transaction.
type PeppleDBTransaction struct {
	db       *PeppleDB
	batch    *pebble.Batch
	cursors  []database.Cursor
	isClosed bool
}

// Begin begins a new transaction.
func (db *PeppleDB) Begin() (database.Transaction, error) {
	batch := db.db.NewBatch()
	transaction := &PeppleDBTransaction{
		db:       db,
		batch:    batch,
		isClosed: false,
	}
	return transaction, nil
}

// Commit commits whatever changes were made to the database within this transaction.
func (tx *PeppleDBTransaction) Commit() error {
	if tx.isClosed {
		return errors.New("cannot commit a closed transaction")
	}
	tx.isClosed = true
	return errors.WithStack(tx.batch.Commit(pebble.Sync))
}

// Rollback rolls back whatever changes were made to the database within this transaction.
func (tx *PeppleDBTransaction) Rollback() error {
	if tx.isClosed {
		return errors.New("cannot rollback a closed transaction")
	}
	tx.isClosed = true
	err := tx.batch.Close()
	return errors.WithStack(err)
}

// RollbackUnlessClosed rolls back changes that were made to the database within the transaction,
// unless the transaction had already been closed using either Rollback or Commit.
func (tx *PeppleDBTransaction) RollbackUnlessClosed() error {
	if tx.isClosed {
		return nil
	}
	return tx.Rollback()
}

// Put sets the value for the given key. It overwrites any previous value for that key.
func (tx *PeppleDBTransaction) Put(key *database.Key, value []byte) error {
	if tx.isClosed {
		return errors.New("cannot put into a closed transaction")
	}
	err := tx.batch.Set(key.Bytes(), value, nil)
	return errors.WithStack(err)
}

// Get gets the value for the given key. It returns ErrNotFound if the given key does not exist.
func (tx *PeppleDBTransaction) Get(key *database.Key) ([]byte, error) {
	if tx.isClosed {
		return nil, errors.New("cannot get from a closed transaction")
	}
	return tx.db.Get(key)
}

// Has returns true if the database contains the given key.
func (tx *PeppleDBTransaction) Has(key *database.Key) (bool, error) {
	if tx.isClosed {
		return false, errors.New("cannot has from a closed transaction")
	}
	return tx.db.Has(key)
}

// Delete deletes the value for the given key. Will not return an error if the key doesn't exist.
func (tx *PeppleDBTransaction) Delete(key *database.Key) error {
	if tx.isClosed {
		return errors.New("cannot delete from a closed transaction")
	}
	err := tx.batch.Delete(key.Bytes(), nil)
	return errors.WithStack(err)
}

// Cursor begins a new cursor over the given bucket.
func (tx *PeppleDBTransaction) Cursor(bucket *database.Bucket) (database.Cursor, error) {
	if tx.isClosed {
		return nil, errors.New("cannot open a cursor from a closed transaction")
	}
	cursor, err := tx.db.Cursor(bucket)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	tx.cursors = append(tx.cursors, cursor)
	return cursor, nil
}
