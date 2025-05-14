package middleware

import (
	"bufio"
	"log"
	"net"
	"net/http"
	"sync"
)

var lrpool = sync.Pool{
	New: func() interface{} {
		return new(logWriter)
	},
}

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

// Size returns the number of bytes currently written in the response.
func (lw *logWriter) Size() int64 {
	return lw.size
}

// Status returns the current response's http status code.
func (lw *logWriter) Status() int {
	return lw.status
}

// Hijack hijacks the current http connection.
func (lw *logWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return lw.ResponseWriter.(http.Hijacker).Hijack()
}

// HandlePanic handles graceful panic by redirecting to friendly error page or rendering a friendly error page.
// trace passed just in case you want rendered to developer when not running in production.
func HandlePanic(w http.ResponseWriter, r *http.Request, trace []byte) {
	// redirect to or directly render friendly error page
}
