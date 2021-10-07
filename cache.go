package gincache

import (
	"sync"
)

// How this cache is implemented:
//
// The main structure is a mapping of keys (string) to cache entries. These entries contain everything
// required to answer a request (status, body & headers) + a set of surrogate keys used for controlled purging.
//
// Entries can be cache in 2 ways: `trySet` will add an entry only if it doesn't exist yet, and forceSet will overwrite
// if necessary.
//
// When evicting we have 3 alternatives:
// - evict everything
// - evict a single entry
// - evict all keys referenced by a surrogate
//
// The first 2 eviction ways are straightforward, but in order to be able to implement the 3rd mechanism,
// we need an extra structure to keep track of which entries are referenced by a certain surrogate.
// We do that with a map of surrogates to a "set" of entry-keys (strings) (implemented as a map to structs{}).
//
// There's a gotcha though:
// Suppose we have the following entry in the cache: `<"e1", ....>` pointed to by surrogate `s1`.
// Suppose we evict `e1` by key and then add a new `e1`. If we purge by `s1`, this entry will be wiped,
// because of it's prior association. In order to avoid this, each entry keeps a list of surrogates that reference it.
// After we wipe a certain key, we need to iterate all those surrogates, and remove any reference to the currently being deleted
// entry.
//
// Below is an example of how these structures reference each other.
//
//
// Cache entries:	(key)	  (status) (body) (headers) (surrogates)
// 		entry1 => {200,    "...", {...},    []}
// 		entry2 => {200,    "...", {...},    []}
// 		entry3 => {200,    "...", {...},    [s1, s2]}
// 		    ^				       |
// Surrogates:	    |				       |
//     s1 ----------|	<------------------------------|
//     s2 ----------|	<------------------------------|

type responseHeaders = map[string]string

type surrogateKeySet = map[string]map[string]struct{}

type cacheEntry struct {
	status     int
	body       []byte
	headers    responseHeaders
	surrogates []string
	sticky     bool
}

type cache interface {
	evictAll()
	evict(entry string)
	evictBySurrogate(key string)
	trySet(entry string, surrogates []string, status int, value []byte, headers responseHeaders, sticky bool) bool
	forceSet(entry string, surrogates []string, status int, value []byte, headers responseHeaders, sticky bool)
	get(entry string) (status int, body []byte, headers responseHeaders)
}

type mtxCache struct {
	max        int
	data       map[string]cacheEntry
	surrogates map[string]map[string]struct{}
	mutex      sync.RWMutex
}

func newMtxCache(size int) *mtxCache {
	return &mtxCache{
		data:       make(map[string]cacheEntry, size),
		surrogates: make(surrogateKeySet),
		max:        size,
	}
}

func (c *mtxCache) evictAll() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data = make(map[string]cacheEntry)
}

func (c *mtxCache) evict(entry string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, entry)
}

func (c *mtxCache) evictBySurrogate(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	entries, _ := c.surrogates[key]
	for entry := range entries {
		current, ok := c.data[entry]

		// We need to iterate through all surrogate keys that point to these entry and remove it from them,
		// otherwise if another key with the same name is added, it could be incorrectly flushed
		var referencingSurrogates []string
		if ok {
			referencingSurrogates = current.surrogates
		}
		delete(c.data, entry)

		for _, referrer := range referencingSurrogates {
			surrogate, ok := c.surrogates[referrer]
			if ok {
				delete(surrogate, referrer)
				c.surrogates[referrer] = surrogate
			}
		}
	}

	// after deleting all the entries associated to the surrogate key, we also delete the surrogate
	delete(c.surrogates, key)
}

func (c *mtxCache) trySet(key string, surrogates []string, status int, body []byte, headers responseHeaders, sticky bool) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.data[key]; ok {
		return false
	}

	if len(c.data) >= c.max {
		c.makeRoom()
	}

	c.data[key] = cacheEntry{status, body, headers, surrogates, sticky}
	c.updateSurrogates(key, surrogates)
	return true
}

func (c *mtxCache) forceSet(key string, surrogates []string, status int, value []byte, headers responseHeaders, sticky bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.data) >= c.max {
		c.makeRoom()
	}

	c.data[key] = cacheEntry{status, value, headers, surrogates, sticky}
	c.updateSurrogates(key, surrogates)
}

func (c *mtxCache) get(key string) (status int, body []byte, headers responseHeaders) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if res, ok := c.data[key]; ok {
		return res.status, res.body, res.headers
	}
	return 0, nil, nil
}

// -- internal

func (c *mtxCache) updateSurrogates(key string, surrogates []string) {
	for _, s := range surrogates {
		current, ok := c.surrogates[s]
		if !ok {
			current = make(map[string]struct{})
		}
		current[key] = struct{}{}
		c.surrogates[s] = current
	}
}

func (c *mtxCache) makeRoom() {
	for k := range c.data {
		if c.data[k].sticky {
			continue
		}
		delete(c.data, k)
		return
	}

	// we did not find a non-sticky entry. delete any. BTW, this is unlikely and mostly bugged
	for k := range c.data {
		delete(c.data, k)
		return
	}

}
