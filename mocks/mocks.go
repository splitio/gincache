package mocks

// CacheFlusherMock implements the CacheFlusher interface
type CacheFlusherMock struct {
	EvictAllCall         func()
	EvictCall            func(key string)
	EvictBySurrogateCall func(surrogate string)
}

// EvictAll clears all the cahed entries
func (c *CacheFlusherMock) EvictAll() {
	c.EvictAllCall()
}

// Evict a single entry
func (c *CacheFlusherMock) Evict(key string) {
	c.EvictCall(key)
}

// EvictBySurrogate keys referenced by a surrogate
func (c *CacheFlusherMock) EvictBySurrogate(key string) {
	c.EvictBySurrogateCall(key)
}
