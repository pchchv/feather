package gzip

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pchchv/feather"
	"github.com/pchchv/feather/assert"
)

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func newCloseNotifyingRecorder() *closeNotifyingRecorder {
	return &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
}

func (c *closeNotifyingRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	reader := bufio.NewReader(c.Body)
	writer := bufio.NewWriter(c.Body)
	return nil, bufio.NewReadWriter(reader, writer), nil
}

func (c *closeNotifyingRecorder) Close() {
	c.closed <- true
}

func (c *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return c.closed
}

func TestGzipFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	buff := new(bytes.Buffer)
	w := gzip.NewWriter(buff)
	gw := gzipWriter{Writer: w, ResponseWriter: rec}
	assert.Equal(t, buff.Len(), 0)

	err := gw.Flush()
	assert.Equal(t, err, nil)

	n1 := buff.Len()
	assert.NotEqual(t, n1, 0)

	_, err = gw.Write([]byte("x"))
	assert.Equal(t, err, nil)

	n2 := buff.Len()
	assert.Equal(t, n1, n2)

	err = gw.Flush()
	assert.Equal(t, err, nil)
	assert.NotEqual(t, n2, buff.Len())
}

func TestGzipHijack(t *testing.T) {
	rec := newCloseNotifyingRecorder()
	buf := new(bytes.Buffer)
	w := gzip.NewWriter(buf)
	gw := gzipWriter{Writer: w, ResponseWriter: rec}
	_, bufrw, err := gw.Hijack()
	assert.Equal(t, err, nil)

	_, _ = bufrw.WriteString("test")
}

func TestGzip(t *testing.T) {
	p := feather.New()
	p.Use(Gzip)
	p.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("test"))
	})
	p.Get("/empty", func(w http.ResponseWriter, r *http.Request) {
	})

	server := httptest.NewServer(p.Serve())
	defer server.Close()

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/test", nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusOK)

	b, err := io.ReadAll(resp.Body)
	assert.Equal(t, err, nil)
	assert.Equal(t, string(b), "test")

	req, _ = http.NewRequest(http.MethodGet, server.URL+"/test", nil)
	req.Header.Set(acceptEncodingHeader, "gzip")
	resp, err = client.Do(req)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
	assert.Equal(t, resp.Header.Get(contentEncodingHeader), gzipVal)
	assert.Equal(t, resp.Header.Get(contentTypeHeader), textPlain)

	r, err := gzip.NewReader(resp.Body)
	assert.Equal(t, err, nil)
	defer r.Close()

	b, err = io.ReadAll(r)
	assert.Equal(t, err, nil)
	assert.Equal(t, string(b), "test")

	req, _ = http.NewRequest(http.MethodGet, server.URL+"/empty", nil)
	resp, err = client.Do(req)
	assert.Equal(t, err, nil)
	assert.Equal(t, resp.StatusCode, http.StatusOK)
}
