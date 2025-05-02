package feather

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
)

var (
	defaultHandler = func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(r.Method)); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
	defaultMiddleware = func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			next(w, r)
		}
	}
)

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func (c *closeNotifyingRecorder) Close() {
	c.closed <- true
}

func (c *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return c.closed
}

func request(method, path string, p *Mux) (int, string) {
	r, _ := http.NewRequest(method, path, nil)
	w := &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
	hf := p.Serve()
	hf.ServeHTTP(w, r)
	return w.Code, w.Body.String()
}

func requestMultiPart(method string, url string, p *Mux) (int, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		fmt.Println("ERR FILE:", err)
	}

	buff := bytes.NewBufferString("FILE TEST DATA")
	if _, err = io.Copy(part, buff); err != nil {
		fmt.Println("ERR COPY:", err)
	}

	if err = writer.WriteField("username", "pchchv"); err != nil {
		fmt.Println("ERR:", err)
	}

	if err = writer.Close(); err != nil {
		fmt.Println("ERR:", err)
	}

	r, _ := http.NewRequest(method, url, body)
	r.Header.Set(contentTypeHeader, writer.FormDataContentType())
	wr := &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
	hf := p.Serve()
	hf.ServeHTTP(wr, r)
	return wr.Code, wr.Body.String()
}
