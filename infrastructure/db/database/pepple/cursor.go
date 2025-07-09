package pepple

import (
	"bytes"

	"github.com/Hoosat-Oy/HTND/infrastructure/db/database"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
)

// PeppleDBCursor is a thin wrapper around Pebble iterators.
type PeppleDBCursor struct {
	iterator *pebble.Iterator
	bucket   *database.Bucket
	isClosed bool
}

// Cursor begins a new cursor over the given prefix.
func (db *PeppleDB) Cursor(bucket *database.Bucket) (database.Cursor, error) {
	// log.Infof("Bucket path = %x", bucket.Path())
	iterator, err := db.db.NewIter(&pebble.IterOptions{
		LowerBound: bucket.Path(),
		// No UpperBound; rely on HasPrefix checks
	})
	return &PeppleDBCursor{
		iterator: iterator,
		bucket:   bucket,
		isClosed: false,
	}, err
}

// Next moves the iterator to the next key/value pair. It returns whether the iterator is exhausted.
// Panics if the cursor is closed.
func (c *PeppleDBCursor) Next() bool {
	if c.isClosed {
		panic("cannot call next on a closed cursor")
	}
	if !c.iterator.Next() {
		return false
	}
	currentKey := c.iterator.Key()
	if currentKey == nil || !bytes.HasPrefix(currentKey, c.bucket.Path()) {
		log.Infof("Next key %x does not match bucket prefix %x", currentKey, c.bucket.Path())
		return false
	}
	log.Infof("Next key: %x", currentKey)
	return true
}

// First moves the iterator to the first key/value pair. It returns false if such a pair does not exist.
// Panics if the cursor is closed.
func (c *PeppleDBCursor) First() bool {
	if c.isClosed {
		panic("cannot call First on a closed cursor")
	}
	if !c.iterator.First() {
		return false
	}
	currentKey := c.iterator.Key()
	if currentKey == nil || !bytes.HasPrefix(currentKey, c.bucket.Path()) {
		log.Infof("First key %x does not match bucket prefix %x", currentKey, c.bucket.Path())
		return false
	}
	log.Infof("First key: %x", currentKey)
	return true
}

// First moves the iterator to the first key/value pair. It returns false if such a pair does not exist.
// Panics if the cursor is closed.
func (c *PeppleDBCursor) Seek(key *database.Key) error {
	if c.isClosed {
		return errors.New("cannot seek a closed cursor")
	}
	fullKey := append(c.bucket.Path(), key.Bytes()...)
	log.Infof("Seeking key: %x (bucket: %x, suffix: %x)", fullKey, c.bucket.Path(), key.Bytes())
	found := c.iterator.SeekGE(fullKey)
	if !found {
		log.Infof("Seek failed: key %x not found", fullKey)
		return errors.Wrapf(database.ErrNotFound, "key %s not found", key)
	}
	currentKey := c.iterator.Key()
	if currentKey == nil || !bytes.HasPrefix(currentKey, c.bucket.Path()) || !bytes.Equal(currentKey, fullKey) {
		log.Infof("Seek mismatch: current key %x, expected %x", currentKey, fullKey)
		return errors.Wrapf(database.ErrNotFound, "key %s not found", key)
	}
	log.Infof("Seek successful: key %x", currentKey)
	return nil
}

// Key returns the key of the current key/value pair, or ErrNotFound if done.
// Note that the key is trimmed to not include the prefix the cursor was opened with.
// The caller should not modify the contents of the returned slice, and its contents may change on the next call to Next.
func (c *PeppleDBCursor) Key() (*database.Key, error) {
	if c.isClosed {
		return nil, errors.New("cannot get the key of a closed cursor")
	}
	fullKeyPath := c.iterator.Key()
	if fullKeyPath == nil {
		return nil, errors.Wrapf(database.ErrNotFound, "cannot get the key of an exhausted cursor")
	}
	if !bytes.HasPrefix(fullKeyPath, c.bucket.Path()) {
		log.Infof("Key %x does not match bucket prefix %x", fullKeyPath, c.bucket.Path())
		return nil, errors.Wrapf(database.ErrNotFound, "key does not match bucket prefix")
	}
	suffix := bytes.TrimPrefix(fullKeyPath, c.bucket.Path())
	log.Infof("Full key: %x, Bucket path: %x, Suffix: %x", fullKeyPath, c.bucket.Path(), suffix)
	return c.bucket.Key(suffix), nil
}

// Value returns the value of the current key/value pair, or ErrNotFound if done.
// The caller should not modify the contents of the returned slice, and its contents may change on the next call to Next.
func (c *PeppleDBCursor) Value() ([]byte, error) {
	if c.isClosed {
		return nil, errors.New("cannot get the value of a closed cursor")
	}
	value := c.iterator.Value()
	if value == nil {
		return nil, errors.Wrapf(database.ErrNotFound, "cannot get the value of an exhausted cursor")
	}
	return value, nil
}

// Close releases associated resources.
func (c *PeppleDBCursor) Close() error {
	if c.isClosed {
		return errors.New("cannot close an already closed cursor")
	}
	c.isClosed = true
	err := c.iterator.Close()
	c.iterator = nil
	c.bucket = nil
	return errors.WithStack(err)
}
