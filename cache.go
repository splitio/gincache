package main

import (
	"sync"
)

type responseHeaders = map[string]string

type cacheEntry struct {
	status  int
	body    []byte
	headers responseHeaders
}

type cache interface {
	evict(entry string)
	trySet(entry string, status int, value []byte, headers responseHeaders) bool
	forceSet(entry string, status int, value []byte, headers responseHeaders)
	get(entry string) (status int, body []byte, headers responseHeaders)
}

type mtxCache struct {
	data  map[string]cacheEntry
	mutex sync.RWMutex
}

func newMtxCache() *mtxCache {
	return &mtxCache{data: make(map[string]cacheEntry)}
}

func (c *mtxCache) evict(entry string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, entry)
}

func (c *mtxCache) trySet(entry string, status int, body []byte, headers responseHeaders) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if _, ok := c.data[entry]; ok {
		return false
	}

	c.data[entry] = cacheEntry{status, body, headers}
	return true
}

func (c *mtxCache) forceSet(entry string, status int, value []byte, headers responseHeaders) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.data[entry] = cacheEntry{status, value, headers}
}

func (c *mtxCache) get(entry string) (status int, body []byte, headers responseHeaders) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if res, ok := c.data[entry]; ok {
		return res.status, res.body, res.headers
	}
	return 0, nil, nil
}
