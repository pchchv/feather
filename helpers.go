package feather

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
)

const (
	UTF8                   = "utf-8"
	charsetUTF8            = "; charset=" + UTF8
	textMarkdown           = textMarkdownNoCharset + charsetUTF8
	textMarkdownNoCharset  = "text/markdown"
	applicationOctetStream = "application/octet-stream"
)

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

// attachment is a helper method for returning an attachment file
// to be downloaded, if you with to open inline see function Inline
func attachment(w http.ResponseWriter, r io.Reader, filename string) (err error) {
	w.Header().Set(contentDispositionHeader, "attachment;filename="+filename)
	w.Header().Set(contentTypeHeader, detectContentType(filename))
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, r)
	return
}
