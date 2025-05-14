package gzip

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"sync"
)

var gzipPool = sync.Pool{
	New: func() interface{} {
		return &gzipWriter{Writer: gzip.NewWriter(io.Discard)}
	},
}

type gzipWriter struct {
	io.Writer
	http.ResponseWriter
	sniffComplete bool
}

func (w *gzipWriter) Flush() error {
	return w.Writer.(*gzip.Writer).Flush()
}

func (w *gzipWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func (w *gzipWriter) Write(b []byte) (int, error) {
	if !w.sniffComplete {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}

		w.sniffComplete = true
	}

	return w.Writer.Write(b)
}
