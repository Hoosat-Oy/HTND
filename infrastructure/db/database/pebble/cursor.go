package pebble

import (
	"bytes"

	"github.com/Hoosat-Oy/HTND/infrastructure/db/database"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
)

// PebbleDBCursor is a thin wrapper around Pebble iterators.
type PebbleDBCursor struct {
	db       *PebbleDB
	iterator *pebble.Iterator
	bucket   *database.Bucket
	isClosed bool
}

// BytesPrefix returns iterator options for keys with the given prefix, with a computed upper bound.
func BytesPrefix(prefix []byte) *pebble.IterOptions {
	var limit []byte
	for i := len(prefix) - 1; i >= 0; i-- {
		c := prefix[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, prefix)
			limit[i] = c + 1
			break
		}
	}

	if limit != nil && 32 > 0 {
		extension := bytes.Repeat([]byte{0xFF}, 32)
		limit = append(limit, extension...)
	}
	// log.Infof("BytesPrefix: prefix=%x, limit=%x", prefix, limit)
	return &pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: limit,
	}
}

// Cursor begins a new cursor over the given prefix.
func (db *PebbleDB) Cursor(bucket *database.Bucket) (database.Cursor, error) {
	// log.Infof("Bucket path = %x", bucket.Path())
	iterator, err := db.db.NewIter(BytesPrefix(bucket.Path()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create iterator")
	}
	cursor := &PebbleDBCursor{
		db:       db,
		iterator: iterator,
		bucket:   bucket,
		isClosed: false,
	}
	db.registerCursor(cursor) // Register cursor with database
	cursor.First()
	return cursor, nil
}

// Next moves the iterator to the next key/value pair. It returns whether the iterator is exhausted.
// Panics if the cursor is closed.
func (c *PebbleDBCursor) Next() bool {
	if c.isClosed {
		panic("cannot call next on a closed cursor")
	}
	// log.Infof("Before Next: valid=%v, key=%x", c.iterator.Valid(), c.iterator.Key())
	hasNext := c.iterator.Next()
	// log.Infof("After Next: hasNext=%v, valid=%v, key=%x", hasNext, c.iterator.Valid(), c.iterator.Key())
	return hasNext
}

// First moves the iterator to the first key/value pair. It returns false if such a pair does not exist.
// Panics if the cursor is closed.
func (c *PebbleDBCursor) First() bool {
	if c.isClosed {
		panic("cannot call First on a closed cursor")
	}
	hasFirst := c.iterator.First()
	// log.Infof("First: hasFirst=%v, currentKey=%x", hasFirst, c.iterator.Key())
	return hasFirst
}

// Seek moves the iterator to the first key/value pair whose key is greater
// than or equal to the given key. It returns ErrNotFound if such pair does not exist.
func (c *PebbleDBCursor) Seek(key *database.Key) error {
	if c.isClosed {
		return errors.New("cannot seek a closed cursor")
	}
	// Use key directly, like LevelDB, for compatibility with UTXO iterator
	found := c.iterator.SeekGE(key.Bytes())
	// log.Infof("Seek: key=%x, found=%v, currentKey=%x", key.Bytes(), found, c.iterator.Key())
	if !found {
		return errors.Wrapf(database.ErrNotFound, "key %s not found", key)
	}
	currentKey := c.iterator.Key()
	if currentKey == nil || !bytes.Equal(currentKey, key.Bytes()) {
		return errors.Wrapf(database.ErrNotFound, "key %s not found", key)
	}
	return nil
}

// Key returns the key of the current key/value pair, or ErrNotFound if done.
// Note that the key is trimmed to not include the prefix the cursor was opened with.
// The caller should not modify the contents of the returned slice, and its contents may change
// on the next call to Next.
func (c *PebbleDBCursor) Key() (*database.Key, error) {
	if c.isClosed {
		return nil, errors.New("cannot get the key of a closed cursor")
	}
	fullKeyPath := c.iterator.Key()
	if fullKeyPath == nil {
		return nil, errors.Wrapf(database.ErrNotFound, "cannot get the key of an exhausted cursor")
	}
	suffix := bytes.TrimPrefix(fullKeyPath, c.bucket.Path())
	// log.Infof("Key: fullKeyPath=%x, suffix=%x", fullKeyPath, suffix)
	return c.bucket.Key(suffix), nil
}

// Value returns the value of the current key/value pair, or ErrNotFound if done.
// The caller should not modify the contents of the returned slice, and its contents may change
// on the next call to Next.
func (c *PebbleDBCursor) Value() ([]byte, error) {
	if c.isClosed {
		return nil, errors.New("cannot get the value of a closed cursor")
	}
	value := c.iterator.Value()
	if value == nil {
		return nil, errors.Wrapf(database.ErrNotFound, "cannot get the value of an exhausted cursor")
	}
	// log.Infof("Value: value=%x", value)
	return value, nil
}

// Close releases associated resources.
func (c *PebbleDBCursor) Close() error {
	if c.isClosed {
		return errors.New("cannot close an already closed cursor")
	}
	c.isClosed = true
	c.db.deregisterCursor(c) // Deregister from database
	err := c.iterator.Close()
	c.iterator = nil
	c.bucket = nil
	return errors.WithStack(err)
}
