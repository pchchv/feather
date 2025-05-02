package feather

import (
	"compress/gzip"
	"io"
	"net/http"
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

func TestNoRequestVars(t *testing.T) {
	reqVars := func(w http.ResponseWriter, r *http.Request) {
		RequestVars(r)
	}
	p := New()
	p.Get("/home", reqVars)
	code, _ := request(http.MethodGet, "/home", p)
	Equal(t, code, http.StatusOK)
}
