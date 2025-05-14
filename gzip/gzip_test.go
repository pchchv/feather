package gzip

import (
	"bufio"
	"net"
	"net/http/httptest"
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
