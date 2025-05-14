package middleware

import (
	"log"
	"net/http"
)

type logWriter struct {
	http.ResponseWriter
	status    int
	size      int64
	committed bool
}

// Write writes the data to the connection as part of an HTTP reply.
// If WriteHeader has not yet been called,
// Write calls WriteHeader(http.StatusOK) before writing the data.
// If the Header does not contain a Content-Type line,
// Write adds a Content-Type set to the result of passing the
// initial 512 bytes of written data to DetectContentType.
func (lw *logWriter) Write(b []byte) (int, error) {
	lw.size += int64(len(b))
	return lw.ResponseWriter.Write(b)
}

// WriteHeader writes HTTP status code.
// If WriteHeader is not called explicitly,
// the first call to Write will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to send error codes.
func (lw *logWriter) WriteHeader(status int) {
	if lw.committed {
		log.Println("response already committed")
		return
	}

	lw.status = status
	lw.ResponseWriter.WriteHeader(status)
	lw.committed = true
}
