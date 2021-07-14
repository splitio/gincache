package gincache

import (
	"bytes"

	"github.com/gin-gonic/gin"
)

type cacheWriter struct {
	gin.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (w *cacheWriter) writeResponse() {
	w.ResponseWriter.Write(w.body.Bytes())
	w.body.Reset()
}

func (w *cacheWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *cacheWriter) Write(data []byte) (n int, err error) {
	return w.body.Write(data)
}

func (w *cacheWriter) WriteString(data string) (n int, err error) {
	return w.body.WriteString(data)
}

func (w *cacheWriter) Size() int {
	return w.body.Len()
}
