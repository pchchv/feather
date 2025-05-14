package gzip

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
)

const (
	contentEncodingHeader = "Content-Encoding"
	acceptEncodingHeader  = "Accept-Encoding"
	contentTypeHeader     = "Content-Type"
	varyHeader            = "Vary"
	gzipVal               = "gzip"
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
		if w.Header().Get(contentTypeHeader) == "" {
			w.Header().Set(contentTypeHeader, http.DetectContentType(b))
		}

		w.sniffComplete = true
	}

	return w.Writer.Write(b)
}

// Gzip returns a middleware which compresses HTTP response using gzip compression scheme.
func Gzip(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add(varyHeader, acceptEncodingHeader)
		if strings.Contains(r.Header.Get(acceptEncodingHeader), gzipVal) {
			gz := gzipPool.Get().(*gzipWriter)
			gz.sniffComplete = false
			gzr := gz.Writer.(*gzip.Writer)
			gzr.Reset(w)
			gz.ResponseWriter = w
			w.Header().Set(contentEncodingHeader, gzipVal)
			w = gz
			defer func() {
				if !gz.sniffComplete {
					// it is necessary to reset response to its
					// pristine state where nothing is written to the body
					w.Header().Del(contentEncodingHeader)
					gzr.Reset(io.Discard)
				}

				gzr.Close()
				gzipPool.Put(gz)
			}()
		}

		next(w, r)
	}
}
