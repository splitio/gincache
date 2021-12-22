package gincache

import (
	"github.com/gin-gonic/gin"
)

var headersToIgnore map[string]struct{} = map[string]struct{}{
	"Access-Control-Allow-Credentials": struct{}{},
	"Access-Control-Expose-Headers":    struct{}{},
	"Access-Control-Allow-Origin":      struct{}{},
	"Vary":                             struct{}{},
}

// StickyEntry is the name of the context key to be set when we want an entry to be sticky
// `sticky` entries are not purged when the cache is full. Only when we forcefully evict them
const StickyEntry = "c_sticky"

// KeyFactoryFn defines the function signature for a Key Factory
type KeyFactoryFn func(ctx *gin.Context) string

// SurrogateFactoryFn defines the function signature for a Surrogate key list factory
type SurrogateFactoryFn func(ctx *gin.Context) []string

// Middleware struct implements a gin middleware that offers request-caching
type Middleware struct {
	keyFactory        KeyFactoryFn
	surrogatesFactory SurrogateFactoryFn
	requestCache      cache
	successOnly       bool
}

// CacheFlusher defines the interface to be used by components that need to evict/flush entries from the cache
type CacheFlusher interface {
	EvictAll()
	Evict(key string)
	EvictBySurrogate(surrogate string)
}

// Options wraps all parameters used to configure the caching middleware
type Options struct {
	Size             int
	KeyFactory       KeyFactoryFn
	SurrogateFactory SurrogateFactoryFn
	SuccessfulOnly   bool
}

// New creates a new middleware with a custom key factory function
func New(options *Options) *Middleware {
	return &Middleware{
		keyFactory:        options.KeyFactory,
		surrogatesFactory: options.SurrogateFactory,
		successOnly:       options.SuccessfulOnly,
		requestCache:      newMtxCache(options.Size),
	}
}

// Handle is the function that should be passed to your router's `.Use()` method
func (h *Middleware) Handle(ctx *gin.Context) {

	if ctx.Request.Method == "OPTIONS" {
		return
	}

	entry := h.keyFactory(ctx)
	if status, response, headers := h.requestCache.get(entry); response != nil {
		for k := range headers {
			if _, shouldIgnore := headersToIgnore[k]; shouldIgnore {
				continue
			}
			ctx.Writer.Header().Add(k, headers[k])
		}
		ctx.Writer.WriteHeader(status)
		ctx.Writer.Write(response)
		ctx.Abort()
		return
	}

	// Setup a writer that intercepts calls made in the req handler and accumulates the response & status code
	originalWriter := ctx.Writer
	withCacheWriter := &cacheWriter{ResponseWriter: originalWriter}
	ctx.Writer = withCacheWriter

	// call the rest of the middleware chain
	ctx.Next()

	if h.successOnly && withCacheWriter.statusCode != 200 {
		return
	}

	headers := make(responseHeaders)
	for k := range withCacheWriter.Header() {
		headers[k] = withCacheWriter.Header().Get(k)
	}

	var surrogates []string
	if h.surrogatesFactory != nil {
		surrogates = h.surrogatesFactory(ctx)
	}

	sticky := ctx.GetBool(StickyEntry)

	// keep a copy of the body for storing in cache
	forCache := make([]byte, len(withCacheWriter.body.Bytes()))
	copy(forCache, withCacheWriter.body.Bytes())
	withCacheWriter.writeResponse()
	h.requestCache.trySet(entry, surrogates, withCacheWriter.statusCode, forCache, headers, sticky)

}

// EvictAll clears all the cahed entries
func (h *Middleware) EvictAll() {
	h.requestCache.evictAll()
}

// Evict a single entry
func (h *Middleware) Evict(key string) {
	h.requestCache.evict(key)
}

// EvictBySurrogate keys referenced by a surrogate
func (h *Middleware) EvictBySurrogate(key string) {
	h.requestCache.evictBySurrogate(key)
}

var _ CacheFlusher = (*Middleware)(nil)
