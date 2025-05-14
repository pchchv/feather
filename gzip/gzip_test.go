package gzip

import "net/http/httptest"

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}
