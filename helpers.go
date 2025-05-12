package feather

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/pchchv/form"
)

const (
	UTF8                   = "utf-8"
	charsetUTF8            = "; charset=" + UTF8
	textMarkdown           = textMarkdownNoCharset + charsetUTF8
	textMarkdownNoCharset  = "text/markdown"
	applicationOctetStream = "application/octet-stream"
)

var DefaultFormEncoder FormEncoder = form.NewEncoder()

// FormEncoder is the type used for encoding form data
type FormEncoder interface {
	Encode(interface{}) (url.Values, error)
}

// RequestVars returns the request scoped variables tracked by feather.
func RequestVars(r *http.Request) ReqVars {
	rv := r.Context().Value(defaultContextIdentifier)
	if rv == nil {
		return new(requestVars)
	}

	return rv.(*requestVars)
}

// ParseForm calls the underlying http.Request ParseForm but also adds the
// URL params to the request Form as if they were defined as query params
// i.e. ?id=13&ok=true but does not add the params to the
// http.Request.URL.RawQuery for SEO purposes.
func ParseForm(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	if rvi := r.Context().Value(defaultContextIdentifier); rvi != nil {
		if rv := rvi.(*requestVars); !rv.formParsed {
			for _, p := range rv.params {
				r.Form.Add(p.key, p.value)
			}
			rv.formParsed = true
		}
	}

	return nil
}

// ParseMultipartForm calls the underlying http.Request ParseMultipartForm but also adds the
// URL params to the request Form as if they were defined as query params
// i.e. ?id=13&ok=true but does not add the params to the
// http.Request.URL.RawQuery for SEO purposes.
func ParseMultipartForm(r *http.Request, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return err
	}

	if rvi := r.Context().Value(defaultContextIdentifier); rvi != nil {
		if rv := rvi.(*requestVars); !rv.formParsed {
			for _, p := range rv.params {
				r.Form.Add(p.key, p.value)
			}
			rv.formParsed = true
		}
	}

	return nil
}

// Attachment is a helper method for returning an attachement file to be downloaded,
// if a line needs to be opened, see the Inline function.
func Attachment(w http.ResponseWriter, r io.Reader, filename string) (err error) {
	w.Header().Set(contentDispositionHeader, "attachment;filename="+filename)
	w.Header().Set(contentTypeHeader, detectContentType(filename))
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, r)
	return
}

// AcceptedLanguages returns an array of accepted languages denoted by
// the Accept-Language header sent by the browser.
func AcceptedLanguages(r *http.Request) (languages []string) {
	accepted := r.Header.Get(acceptedLanguageHeader)
	if accepted == "" {
		return
	}

	options := strings.Split(accepted, ",")
	l := len(options)
	languages = make([]string, l)
	for i := 0; i < l; i++ {
		locale := strings.SplitN(options[i], ";", 2)
		languages[i] = strings.Trim(locale[0], " ")
	}

	return
}

// EncodeToURLValues encodes a struct or field into a set of url.Values.
func EncodeToURLValues(v interface{}) (url.Values, error) {
	return DefaultFormEncoder.Encode(v)
}

func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}

	switch ext {
	case ".md":
		return textMarkdown
	default:
		return applicationOctetStream
	}
}
