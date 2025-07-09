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
	// Use LowerBound and UpperBound to restrict iterator to bucket prefix
	iterator, err := db.db.NewIter(&pebble.IterOptions{
		LowerBound: bucket.Path(),
		UpperBound: append(bucket.Path(), 0xff),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &PeppleDBCursor{
		iterator: iterator,
		bucket:   bucket,
		isClosed: false,
	}, nil
}

// Next moves the iterator to the next key/value pair. It returns whether the iterator is exhausted.
// Panics if the cursor is closed.
func (c *PeppleDBCursor) Next() bool {
	if c.isClosed {
		panic("cannot call next on a closed cursor")
	}
	return c.iterator.Next()
}

// First moves the iterator to the first key/value pair. It returns false if such a pair does not exist.
// Panics if the cursor is closed.
func (c *PeppleDBCursor) First() bool {
	if c.isClosed {
		panic("cannot call first on a closed cursor")
	}
	return c.iterator.First()
}

// Seek moves the iterator to the first key/value pair whose key is greater than or equal to the given key.
// It returns ErrNotFound if such pair does not exist.
func (c *PeppleDBCursor) Seek(key *database.Key) error {
	if c.isClosed {
		return errors.New("cannot seek a closed cursor")
	}

	found := c.iterator.SeekGE(key.Bytes())
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
// The caller should not modify the contents of the returned slice, and its contents may change on the next call to Next.
func (c *PeppleDBCursor) Key() (*database.Key, error) {
	if c.isClosed {
		return nil, errors.New("cannot get the key of a closed cursor")
	}
	fullKeyPath := c.iterator.Key()
	if fullKeyPath == nil {
		return nil, errors.Wrapf(database.ErrNotFound, "cannot get the key of an exhausted cursor")
	}
	suffix := bytes.TrimPrefix(fullKeyPath, c.bucket.Path())
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
	// Create a copy of the value to ensure it’s not modified externally
	valueCopy := make([]byte, len(value))
	copy(valueCopy, value)
	return valueCopy, nil
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
