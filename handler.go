package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// Middleware struct implements a gin middleware that offers request-caching
type Middleware struct {
	keyFactory   func(ctx *gin.Context) string
	requestCache cache
}

// New creates a new middleware with a custom key factory function
func New(keyFactory func(ctx *gin.Context) string) *Middleware {
	return &Middleware{keyFactory: keyFactory, requestCache: newMtxCache()}
}

// Handle is the function that should be passed to your router's `.Use()` method
func (h *Middleware) Handle(ctx *gin.Context) {
	entry := h.keyFactory(ctx)
	fmt.Println("Entry: ", entry)
	if status, response, headers := h.requestCache.get(entry); response != nil {
		for k := range headers {
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

	// Schedule a function to be executed after request handling is done
	defer func() {
		headers := make(responseHeaders)
		for k := range withCacheWriter.Header() {
			headers[k] = withCacheWriter.Header().Get(k)
		}
		go h.requestCache.trySet(entry, withCacheWriter.statusCode, withCacheWriter.body.Bytes(), headers)
		withCacheWriter.writeResponse()
	}()

	// call the rest of the middleware chain
	ctx.Next()
}

// EvictAll clears all the cahed entries
func (h *Middleware) EvictAll() {
	h.requestCache.evictAll()
}

// Evict a single entry
func (h *Middleware) Evict(key string) {
	h.requestCache.evict(key)
}
