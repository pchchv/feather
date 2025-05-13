package gzip

import (
	"io"
	"net/http"
)

type gzipWriter struct {
	io.Writer
	http.ResponseWriter
	sniffComplete bool
}
