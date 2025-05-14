package middleware

import "net/http"

type logWriter struct {
	http.ResponseWriter
	status    int
	size      int64
	committed bool
}
