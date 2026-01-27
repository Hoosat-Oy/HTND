package lrucache

import (
	// "sync"

	"github.com/Hoosat-Oy/HTND/domain/consensus/model/externalapi"
)

// LRUCache is a least-recently-used cache for any type
// that's able to be indexed by DomainHash
type LRUCache struct {
	// lock     *sync.RWMutex
	cache    map[externalapi.DomainHash]interface{}
	capacity int
}

// New creates a new LRUCache
func New(capacity int, preallocate bool) *LRUCache {
	var cache map[externalapi.DomainHash]interface{}
	if preallocate {
		cache = make(map[externalapi.DomainHash]interface{}, capacity+1)
	} else {
		cache = make(map[externalapi.DomainHash]interface{})
	}
	return &LRUCache{
		// lock:     &sync.RWMutex{},
		cache:    cache,
		capacity: capacity,
	}
}

// Add adds an entry to the LRUCache
func (c *LRUCache) Add(key *externalapi.DomainHash, value interface{}) {
	// c.lock.Lock()
	// defer c.lock.Unlock()
	c.cache[*key] = value

	if len(c.cache) > c.capacity {
		c.evictRandom()
	}
}

// Get returns the entry for the given key, or (nil, false) otherwise
func (c *LRUCache) Get(key *externalapi.DomainHash) (interface{}, bool) {
	// c.lock.RLock()
	// defer c.lock.RUnlock()
	value, ok := c.cache[*key]
	if !ok {
		return nil, false
	}
	return value, true
}

// Has returns whether the LRUCache contains the given key
func (c *LRUCache) Has(key *externalapi.DomainHash) bool {
	// c.lock.RLock()
	// defer c.lock.RUnlock()
	DomainHash, ok := c.cache[*key]
	return ok && DomainHash != nil
}

// Remove removes the entry for the the given key. Does nothing if
// the entry does not exist
func (c *LRUCache) Remove(key *externalapi.DomainHash) {
	// c.lock.Lock()
	// defer c.lock.Unlock()
	delete(c.cache, *key)
}

func (c *LRUCache) evictRandom() {
	var keyToEvict externalapi.DomainHash
	for key := range c.cache {
		keyToEvict = key
		break
	}
	delete(c.cache, keyToEvict)
}
