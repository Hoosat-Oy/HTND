package pepple

import (
	"os"

	"github.com/Hoosat-Oy/HTND/infrastructure/db/database"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
)

// PeppleDB defines a thin wrapper around Pebble.
type PeppleDB struct {
	db *pebble.DB
}

// NewPeppleDB opens a Pebble instance defined by the given path.
func NewPeppleDB(path string, cacheSizeMiB int) (*PeppleDB, error) {
	options := Options()

	db, err := pebble.Open(path, options)
	if err != nil {
		if errors.Is(err, pebble.ErrCorruption) {
			log.Warnf("Pebble corruption detected at %s: %v", path, err)

			// Remove the corrupted DB
			log.Warnf("Removing corrupted DB at %s", path)
			if rmErr := os.RemoveAll(path); rmErr != nil {
				return nil, errors.Wrap(rmErr, "failed to remove corrupted DB")
			}

			// Attempt to create a fresh DB
			db, err = pebble.Open(path, options)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create fresh DB after corruption")
			}
			log.Warnf("Created fresh Pebble DB at %s", path)
		} else {
			return nil, errors.WithStack(err)
		}
	}

	dbInstance := &PeppleDB{
		db: db,
	}
	return dbInstance, nil
}

// Compact compacts the Pebble instance.
func (db *PeppleDB) Compact() error {
	err := db.db.Compact(nil, []byte{0xff, 0xff, 0xff, 0xff}, false)
	return errors.WithStack(err)
}

// Close closes the Pebble instance.
func (db *PeppleDB) Close() error {
	err := db.db.Close()
	return errors.WithStack(err)
}

// Put sets the value for the given key. It overwrites any previous value for that key.
func (db *PeppleDB) Put(key *database.Key, value []byte) error {
	err := db.db.Set(key.Bytes(), value, pebble.Sync)
	return errors.WithStack(err)
}

// Get gets the value for the given key. It returns ErrNotFound if the given key does not exist.
func (db *PeppleDB) Get(key *database.Key) ([]byte, error) {
	data, closer, err := db.db.Get(key.Bytes())
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, errors.Wrapf(database.ErrNotFound, "key %s not found", key)
		}
		return nil, errors.WithStack(err)
	}
	defer closer.Close()
	return data, nil
}

// Has returns true if the database contains the given key.
func (db *PeppleDB) Has(key *database.Key) (bool, error) {
	_, closer, err := db.db.Get(key.Bytes())
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	defer closer.Close()
	return true, nil
}

// Delete deletes the value for the given key. Will not return an error if the key doesn't exist.
func (db *PeppleDB) Delete(key *database.Key) error {
	err := db.db.Delete(key.Bytes(), pebble.Sync)
	return errors.WithStack(err)
}
