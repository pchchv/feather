package feather

import (
	"bufio"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"

	. "github.com/pchchv/feather/assert"
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

func (w *gzipWriter) Write(b []byte) (int, error) {
	if !w.sniffComplete {
		if w.Header().Get(contentTypeHeader) == "" {
			w.Header().Set(contentTypeHeader, http.DetectContentType(b))
		}
		w.sniffComplete = true
	}

	return w.Writer.Write(b)
}

func (w *gzipWriter) Flush() error {
	return w.Writer.(*gzip.Writer).Flush()
}

func (w *gzipWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
}

func TestNoRequestVars(t *testing.T) {
	reqVars := func(w http.ResponseWriter, r *http.Request) {
		RequestVars(r)
	}
	p := New()
	p.Get("/home", reqVars)
	code, _ := request(http.MethodGet, "/home", p)
	Equal(t, code, http.StatusOK)
}

// Gzip2 returns a middleware which compresses HTTP response using gzip compression scheme.
func Gzip2(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var gz *gzipWriter
		var gzr *gzip.Writer
		w.Header().Add(varyHeader, acceptEncodingHeader)
		if strings.Contains(r.Header.Get(acceptEncodingHeader), "gzip") {
			gz = gzipPool.Get().(*gzipWriter)
			gz.sniffComplete = false
			gzr = gz.Writer.(*gzip.Writer)
			gzr.Reset(w)
			gz.ResponseWriter = w

			w.Header().Set(acceptEncodingHeader, "gzip")

			w = gz
			defer func() {
				if !gz.sniffComplete {
					// have to reset response to it's pristine state when nothing is written to body
					w.Header().Del(acceptEncodingHeader)
					gzr.Reset(io.Discard)
				}

				gzr.Close()
				gzipPool.Put(gz)
			}()
		}

		next(w, r)
	}
}

func TestBadParseForm(t *testing.T) {
	// successful scenarios tested under TestDecode
	p := New()
	p.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		if err := ParseForm(r); err != nil {
			if _, err = w.Write([]byte(err.Error())); err != nil {
				panic(err)
			}
			return
		}
	})

	code, body := request(http.MethodGet, "/users/16?test=%2f%%efg", p)
	Equal(t, code, http.StatusOK)
	Equal(t, body, "invalid URL escape \"%%e\"")
}

func TestBadParseMultiPartForm(t *testing.T) {
	// successful scenarios tested under TestDecode
	p := New()
	p.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		if e := ParseMultipartForm(r, 10<<5); e != nil {
			if _, err := w.Write([]byte(e.Error())); err != nil {
				panic(e)
			}
			return
		}
	})

	code, body := requestMultiPart(http.MethodGet, "/users/16?test=%2f%%efg", p)
	Equal(t, code, http.StatusOK)
	Equal(t, body, "invalid URL escape \"%%e\"")
}
