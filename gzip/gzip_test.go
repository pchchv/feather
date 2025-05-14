package gzip

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"net"
	"net/http/httptest"
	"testing"

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
