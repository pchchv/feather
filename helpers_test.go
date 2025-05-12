package feather

import (
	"bufio"
	"compress/gzip"
	"encoding/xml"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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

func (w *gzipWriter) Write(b []byte) (int, error) {
	if !w.sniffComplete {
		if w.Header().Get(contentTypeHeader) == "" {
			w.Header().Set(contentTypeHeader, http.DetectContentType(b))
		}
		w.sniffComplete = true
	}

	return w.Writer.Write(b)
}

func (w *gzipWriter) Flush() error {
	return w.Writer.(*gzip.Writer).Flush()
}

func (w *gzipWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.(http.Hijacker).Hijack()
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

// Gzip2 returns a middleware which compresses HTTP response using gzip compression scheme.
func Gzip2(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var gz *gzipWriter
		var gzr *gzip.Writer
		w.Header().Add(varyHeader, acceptEncodingHeader)
		if strings.Contains(r.Header.Get(acceptEncodingHeader), "gzip") {
			gz = gzipPool.Get().(*gzipWriter)
			gz.sniffComplete = false
			gzr = gz.Writer.(*gzip.Writer)
			gzr.Reset(w)
			gz.ResponseWriter = w

			w.Header().Set(acceptEncodingHeader, "gzip")

			w = gz
			defer func() {
				if !gz.sniffComplete {
					// have to reset response to it's pristine state when nothing is written to body
					w.Header().Del(acceptEncodingHeader)
					gzr.Reset(io.Discard)
				}

				gzr.Close()
				gzipPool.Put(gz)
			}()
		}

		next(w, r)
	}
}

func TestBadParseForm(t *testing.T) {
	// successful scenarios tested under TestDecode
	p := New()
	p.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		if err := ParseForm(r); err != nil {
			if _, err = w.Write([]byte(err.Error())); err != nil {
				panic(err)
			}
			return
		}
	})

	code, body := request(http.MethodGet, "/users/16?test=%2f%%efg", p)
	Equal(t, code, http.StatusOK)
	Equal(t, body, "invalid URL escape \"%%e\"")
}

func TestBadParseMultiPartForm(t *testing.T) {
	// successful scenarios tested under TestDecode
	p := New()
	p.Get("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		if e := ParseMultipartForm(r, 10<<5); e != nil {
			if _, err := w.Write([]byte(e.Error())); err != nil {
				panic(e)
			}
			return
		}
	})

	code, body := requestMultiPart(http.MethodGet, "/users/16?test=%2f%%efg", p)
	Equal(t, code, http.StatusOK)
	Equal(t, body, "invalid URL escape \"%%e\"")
}

func TestAcceptedLanguages(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set(acceptedLanguageHeader, "da, en-GB;q=0.8, en;q=0.7")
	languages := AcceptedLanguages(req)
	Equal(t, languages[0], "da")
	Equal(t, languages[1], "en-GB")
	Equal(t, languages[2], "en")

	req.Header.Del(acceptedLanguageHeader)
	languages = AcceptedLanguages(req)
	Equal(t, len(languages), 0)

	req.Header.Set(acceptedLanguageHeader, "")
	languages = AcceptedLanguages(req)
	Equal(t, len(languages), 0)
}

func TestAttachment(t *testing.T) {
	p := New()
	p.Get("/dl", func(w http.ResponseWriter, r *http.Request) {
		f, _ := os.Open("logo.png")
		if err := Attachment(w, f, "logo.png"); err != nil {
			panic(err)
		}
	})
	p.Get("/dl-unknown-type", func(w http.ResponseWriter, r *http.Request) {
		f, _ := os.Open("logo.png")
		if err := Attachment(w, f, "logo"); err != nil {
			panic(err)
		}
	})
	r, _ := http.NewRequest(http.MethodGet, "/dl", nil)
	w := &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
	hf := p.Serve()
	hf.ServeHTTP(w, r)
	Equal(t, w.Code, http.StatusOK)
	Equal(t, w.Header().Get(contentDispositionHeader), "attachment;filename=logo.png")
	Equal(t, w.Header().Get(contentTypeHeader), "image/png")
	Equal(t, w.Body.Len(), 20797)

	r, _ = http.NewRequest(http.MethodGet, "/dl-unknown-type", nil)
	w = &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
	hf = p.Serve()
	hf.ServeHTTP(w, r)
	Equal(t, w.Code, http.StatusOK)
	Equal(t, w.Header().Get(contentDispositionHeader), "attachment;filename=logo")
	Equal(t, w.Header().Get(contentTypeHeader), "application/octet-stream")
	Equal(t, w.Body.Len(), 20797)
}

func TestEncodeToURLValues(t *testing.T) {
	type Test struct {
		Domain string `form:"domain"`
		Next   string `form:"next"`
	}

	s := Test{Domain: "company.org", Next: "NIDEJ89#(@#NWJK"}
	values, err := EncodeToURLValues(s)
	Equal(t, err, nil)
	Equal(t, len(values), 2)
	Equal(t, values.Encode(), "domain=company.org&next=NIDEJ89%23%28%40%23NWJK")
}

func TestInline(t *testing.T) {
	p := New()
	p.Get("/dl-inline", func(w http.ResponseWriter, r *http.Request) {
		f, _ := os.Open("logo.png")
		if err := Inline(w, f, "logo.png"); err != nil {
			panic(err)
		}
	})
	p.Get("/dl-unknown-type-inline", func(w http.ResponseWriter, r *http.Request) {
		f, _ := os.Open("logo.png")
		if err := Inline(w, f, "logo"); err != nil {
			panic(err)
		}
	})
	r, _ := http.NewRequest(http.MethodGet, "/dl-inline", nil)
	w := &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
	hf := p.Serve()
	hf.ServeHTTP(w, r)
	Equal(t, w.Code, http.StatusOK)
	Equal(t, w.Header().Get(contentDispositionHeader), "inline;filename=logo.png")
	Equal(t, w.Header().Get(contentTypeHeader), "image/png")
	Equal(t, w.Body.Len(), 20797)

	r, _ = http.NewRequest(http.MethodGet, "/dl-unknown-type-inline", nil)
	w = &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
	hf = p.Serve()
	hf.ServeHTTP(w, r)
	Equal(t, w.Code, http.StatusOK)
	Equal(t, w.Header().Get(contentDispositionHeader), "inline;filename=logo")
	Equal(t, w.Header().Get(contentTypeHeader), "application/octet-stream")
	Equal(t, w.Body.Len(), 20797)
}

func TestClientIP(t *testing.T) {
	req, _ := http.NewRequest("POST", "/", nil)
	req.Header.Set("X-Real-IP", " 10.10.10.10  ")
	req.Header.Set("X-Forwarded-For", "  20.20.20.20, 30.30.30.30")
	req.RemoteAddr = "  40.40.40.40:42123 "
	Equal(t, ClientIP(req), "10.10.10.10")

	req.Header.Del("X-Real-IP")
	Equal(t, ClientIP(req), "20.20.20.20")

	req.Header.Set("X-Forwarded-For", "30.30.30.30  ")
	Equal(t, ClientIP(req), "30.30.30.30")

	req.Header.Del("X-Forwarded-For")
	Equal(t, ClientIP(req), "40.40.40.40")
}

func TestXML(t *testing.T) {
	xmlData := `<zombie><id>1</id><name>Patient Zero</name></zombie>`
	p := New()
	p.Use(Gzip2)
	p.Get("/xml", func(w http.ResponseWriter, r *http.Request) {
		if err := XML(w, http.StatusOK, zombie{1, "Patient Zero"}); err != nil {
			panic(err)
		}
	})
	p.Get("/badxml", func(w http.ResponseWriter, r *http.Request) {
		if err := XML(w, http.StatusOK, func() {}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	p.Get("/xmlbytes", func(w http.ResponseWriter, r *http.Request) {
		b, _ := xml.Marshal(zombie{1, "Patient Zero"})
		if err := XMLBytes(w, http.StatusOK, b); err != nil {
			panic(err)
		}
	})

	hf := p.Serve()
	r, _ := http.NewRequest(http.MethodGet, "/xml", nil)
	w := httptest.NewRecorder()
	hf.ServeHTTP(w, r)
	Equal(t, w.Code, http.StatusOK)
	Equal(t, w.Header().Get(contentTypeHeader), applicationXML)
	Equal(t, w.Body.String(), xml.Header+xmlData)

	r, _ = http.NewRequest(http.MethodGet, "/xmlbytes", nil)
	w = httptest.NewRecorder()
	hf.ServeHTTP(w, r)
	Equal(t, w.Code, http.StatusOK)
	Equal(t, w.Header().Get(contentTypeHeader), applicationXML)
	Equal(t, w.Body.String(), xml.Header+xmlData)

	r, _ = http.NewRequest(http.MethodGet, "/badxml", nil)
	w = httptest.NewRecorder()
	hf.ServeHTTP(w, r)
	Equal(t, w.Code, http.StatusInternalServerError)
	Equal(t, w.Header().Get(contentTypeHeader), textPlain)
	Equal(t, w.Body.String(), "xml: unsupported type: func()\n")
}
