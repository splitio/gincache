package main

import (
	"bytes"
	"fmt"

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
	fmt.Println("WriteHeader")
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *cacheWriter) Write(data []byte) (n int, err error) {
	fmt.Println("Write")
	return w.body.Write(data)
}

func (w *cacheWriter) WriteString(data string) (n int, err error) {
	fmt.Println("WriteLn")
	return w.body.WriteString(data)
}

// func (w *cacheWriter) Status() int {
// 	fmt.Println("Status")
// 	return w.ResponseWriter.Status()
// }
//
func (w *cacheWriter) Size() int {
	fmt.Println("Size", w.body.Len())
	return w.body.Len()
	//return w.ResponseWriter.Size()
}

//
// func (w *cacheWriter) Written() bool {
// 	fmt.Println("Written")
// 	return w.ResponseWriter.Written()
// }
//
// // Hijack implements the http.Hijacker interface.
// func (w *cacheWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
// 	fmt.Println("HiJack")
// 	return w.ResponseWriter.Hijack()
// }
//
// // CloseNotify implements the http.CloseNotify interface.
// func (w *cacheWriter) CloseNotify() <-chan bool {
// 	fmt.Println("CloseNotify")
// 	return w.ResponseWriter.CloseNotify()
// }
//
// // Flush implements the http.Flush interface.
// func (w *cacheWriter) Flush() {
// 	fmt.Println("Flush")
// 	w.ResponseWriter.Flush()
// }
//
// func (w *cacheWriter) Pusher() (pusher http.Pusher) {
// 	fmt.Println("Pusher")
// 	return w.ResponseWriter.Pusher()
// }
