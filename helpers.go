package feather

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// QueryParamsOption represents the options for
// including query parameters during Decode helper functions.
type QueryParamsOption uint8

const (
	httpQueryParams QueryParamsOption = iota
	noQueryParams
	applicationForm          = "application/x-www-form-urlencoded"
	applicationOctetStream   = "application/octet-stream"
	applicationJSON          = applicationJSONNoCharset + charsetUTF8
	applicationJSONNoCharset = "application/json"
	applicationXML           = applicationXMLNoCharset + charsetUTF8
	applicationXMLNoCharset  = "application/xml"
	charsetUTF8              = "; charset=" + utf8
	gzipVal                  = "gzip"
	multipartForm            = "multipart/form-data"
	nakedApplicationXML      = "application/xml"
	nakedApplicationJSON     = "application/json"
	textPlain                = textPlainNoCharset + charsetUTF8
	textPlainNoCharset       = "text/plain"
	textMarkdown             = textMarkdownNoCharset + charsetUTF8
	textMarkdownNoCharset    = "text/markdown"
	utf8                     = "utf-8"
)

var xmlHeaderBytes = []byte(xml.Header)

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

// Inline is a helper method for returning a file inline to be rendered/opened by the browser.
func Inline(w http.ResponseWriter, r io.Reader, filename string) (err error) {
	w.Header().Set(contentDispositionHeader, "inline;filename="+filename)
	w.Header().Set(contentTypeHeader, detectContentType(filename))
	w.WriteHeader(http.StatusOK)
	_, err = io.Copy(w, r)
	return
}

// ClientIP implements a best effort algorithm to return the real client IP,
// it parses X-Real-IP and X-Forwarded-For in order to
// work properly with reverse-proxies such us: nginx or haproxy.
func ClientIP(r *http.Request) (clientIP string) {
	values := r.Header[xRealIPHeader]
	if len(values) > 0 {
		clientIP = strings.TrimSpace(values[0])
		if clientIP != "" {
			return
		}
	}

	if values = r.Header[xForwardedForHeader]; len(values) > 0 {
		clientIP = values[0]
		if index := strings.IndexByte(clientIP, ','); index >= 0 {
			clientIP = clientIP[0:index]
		}

		clientIP = strings.TrimSpace(clientIP)
		if clientIP != "" {
			return
		}
	}

	clientIP, _, _ = net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	return
}

// XML marshals provided interface + returns XML + status code.
func XML(w http.ResponseWriter, status int, i interface{}) error {
	b, err := xml.Marshal(i)
	if err != nil {
		return err
	}

	w.Header().Set(contentTypeHeader, applicationXML)
	w.WriteHeader(status)
	if _, err = w.Write(xmlHeaderBytes); err == nil {
		_, err = w.Write(b)
	}

	return err
}

// XMLBytes returns provided XML response with status code.
func XMLBytes(w http.ResponseWriter, status int, b []byte) (err error) {
	w.Header().Set(contentTypeHeader, applicationXML)
	w.WriteHeader(status)
	if _, err = w.Write(xmlHeaderBytes); err == nil {
		_, err = w.Write(b)
	}

	return
}

// JSON marshals provided interface + returns JSON + status code.
func JSON(w http.ResponseWriter, status int, i interface{}) error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}

	w.Header().Set(contentTypeHeader, applicationJSON)
	w.WriteHeader(status)
	_, err = w.Write(b)
	return err
}

// JSONP sends a JSONP response with status code and uses `callback` to construct the JSONP payload.
func JSONP(w http.ResponseWriter, status int, i interface{}, callback string) error {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}

	w.Header().Set(contentTypeHeader, applicationJSON)
	w.WriteHeader(status)
	if _, err = w.Write([]byte(callback + "(")); err == nil {
		if _, err = w.Write(b); err == nil {
			_, err = w.Write([]byte(");"))
		}
	}

	return err
}

// JSONBytes returns provided JSON response with status code.
func JSONBytes(w http.ResponseWriter, status int, b []byte) (err error) {
	w.Header().Set(contentTypeHeader, applicationJSON)
	w.WriteHeader(status)
	_, err = w.Write(b)
	return err
}

// JSONStream uses json.Encoder to stream the JSON reponse body.
//
// This differs from the JSON helper which unmarshalls into memory first allowing the
// capture of JSON encoding errors.
func JSONStream(w http.ResponseWriter, status int, i interface{}) error {
	w.Header().Set(contentTypeHeader, applicationJSON)
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(i)
}

// QueryParams returns the r.URL.Query() values and optionally have them include the
// SEO query params eg. route /users/:id?test=val if qp=QueryParams then
// values will include 'id' as well as 'test' values.
func QueryParams(r *http.Request, qp QueryParamsOption) (values url.Values) {
	values = r.URL.Query()
	if qp == httpQueryParams {
		if rvi := r.Context().Value(defaultContextIdentifier); rvi != nil {
			rv := rvi.(*requestVars)
			for _, p := range rv.params {
				values.Add(p.key, p.value)
			}
		}
	}

	return
}

// DecodeMultipartForm parses the requests form data into the provided struct.
//
// The Content-Type and http method are not checked.
//
// NOTE: when qp=QueryParams, both query parameters and SEO query parameters will be parsed and included,
// e.g. the route /user/:id?test=true both 'id' and 'test' are treated as query parameters and added to request.Form prior to decoding.
// SEO query params are treated just like normal query params.
func DecodeMultipartForm(r *http.Request, qp QueryParamsOption, maxMemory int64, v interface{}) (err error) {
	if qp == httpQueryParams {
		if err = ParseMultipartForm(r, maxMemory); err != nil {
			return
		}
	}

	if err = r.ParseMultipartForm(maxMemory); err == nil {
		switch qp {
		case httpQueryParams:
			err = DefaultFormDecoder.Decode(v, r.Form)
		case noQueryParams:
			err = DefaultFormDecoder.Decode(v, r.MultipartForm.Value)
		}
	}

	return
}

// DecodeSEOQueryParams decodes the SEO Query params only and ignores the normal URL Query params.
func DecodeSEOQueryParams(r *http.Request, v interface{}) (err error) {
	if rvi := r.Context().Value(defaultContextIdentifier); rvi != nil {
		rv := rvi.(*requestVars)
		values := make(url.Values, len(rv.params))
		for _, p := range rv.params {
			values.Add(p.key, p.value)
		}

		err = DefaultFormDecoder.Decode(v, values)
	}

	return
}

// DecodeForm parses the requests form data into the provided struct.
//
// The Content-Type and http method are not checked.
//
// NOTE: when qp=QueryParams, both query parameters and SEO query parameters will be parsed and included,
// e.g. the route /user/:id?test=true both 'id' and 'test' are treated as query parameters and added to request.Form prior to decoding.
// SEO query params are treated just like normal query params.
func DecodeForm(r *http.Request, qp QueryParamsOption, v interface{}) (err error) {
	if qp == httpQueryParams {
		if err = ParseForm(r); err != nil {
			return
		}
	}

	if err = r.ParseForm(); err == nil {
		switch qp {
		case httpQueryParams:
			err = DefaultFormDecoder.Decode(v, r.Form)
		case noQueryParams:
			err = DefaultFormDecoder.Decode(v, r.PostForm)
		}
	}

	return
}

// DecodeXML decodes the request body into the provided struct and limits the
// request size via an ioext.LimitReader using the maxMemory param.
//
// The Content-Type e.g. "application/xml" and http method are not checked.
//
// NOTE: when qp=QueryParams both query params and SEO query params will be parsed and included
// e. g. route /user/:id?test=true both 'id' and 'test' are treated as query params and added to parsed XML.
// SEO query params are treated just like normal query params.
func DecodeXML(r *http.Request, qp QueryParamsOption, maxMemory int64, v interface{}) error {
	var values url.Values
	if qp == httpQueryParams {
		values = r.URL.Query()
	}

	return decodeXML(r.Header, r.Body, qp, values, maxMemory, v)
}

// DecodeJSON decodes the request body into the provided struct and limits the
// request size via an ioext.LimitReader using the maxMemory param.
//
// The Content-Type e.g. "application/json" and http method are not checked.
//
// NOTE: when qp=QueryParams both query params and SEO query params will be parsed and included
// e. g. route /user/:id?test=true both 'id' and 'test' are treated as query params and added to parsed JSON.
// SEO query params are treated just like normal query params.
func DecodeJSON(r *http.Request, qp QueryParamsOption, maxMemory int64, v interface{}) error {
	var values url.Values
	if qp == httpQueryParams {
		values = r.URL.Query()
	}

	return decodeJSON(r.Header, r.Body, qp, values, maxMemory, v)
}

// DecodeQueryParams takes the URL Query params,
// adds SEO params or not based on the includeSEOQueryParams flag.
//
// NOTE: DecodeQueryParams is also used/called from Decode when
// no contentTypeHeader is specified the only difference is that
// it will always decode SEO Query Params.
func DecodeQueryParams(r *http.Request, qp QueryParamsOption, v interface{}) error {
	return DefaultFormDecoder.Decode(v, QueryParams(r, qp))
}

// Decode takes the request and attempts to discover it's content type via the
// http headers and then decode the request body into the provided struct.
// Example if header was "application/json" would decode using json.NewDecoder(ioext.LimitReader(r.Body, maxMemory)).Decode(v).
//
// NOTE: when qp=QueryParams both query params and SEO query params will be parsed and included
// e. g. route /user/:id?test=true both 'id' and 'test' are treated as query params and added to the
// request.Form prior to decoding or added to parsed JSON or XML.
// SEO query params are treated just like normal query params.
func Decode(r *http.Request, qp QueryParamsOption, maxMemory int64, v interface{}) (err error) {
	typ := r.Header.Get(contentTypeHeader)
	if idx := strings.Index(typ, ";"); idx != -1 {
		typ = typ[:idx]
	}

	switch typ {
	case applicationForm:
		err = DecodeForm(r, qp, v)
	case multipartForm:
		err = DecodeMultipartForm(r, qp, maxMemory, v)
	default:
		if qp == httpQueryParams {
			if err = DecodeSEOQueryParams(r, v); err != nil {
				return
			}
		}
		err = decode(r, qp, maxMemory, v)
	}

	return
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

func decodeQueryParams(values url.Values, v interface{}) error {
	return DefaultFormDecoder.Decode(v, values)
}

func decodeXML(headers http.Header, body io.Reader, qp QueryParamsOption, values url.Values, maxMemory int64, v interface{}) (err error) {
	if encoding := headers.Get(contentEncodingHeader); encoding == gzipVal {
		var gzr *gzip.Reader
		gzr, err = gzip.NewReader(body)
		if err != nil {
			return
		}

		defer func() {
			_ = gzr.Close()
		}()

		body = gzr
	}

	err = xml.NewDecoder(LimitReader(body, maxMemory)).Decode(v)
	if qp == httpQueryParams && err == nil {
		err = decodeQueryParams(values, v)
	}

	return
}

func decodeJSON(headers http.Header, body io.Reader, qp QueryParamsOption, values url.Values, maxMemory int64, v interface{}) (err error) {
	if encoding := headers.Get(contentEncodingHeader); encoding == gzipVal {
		var gzr *gzip.Reader
		gzr, err = gzip.NewReader(body)
		if err != nil {
			return
		}

		defer func() {
			_ = gzr.Close()
		}()
		body = gzr
	}

	err = json.NewDecoder(LimitReader(body, maxMemory)).Decode(v)
	if qp == httpQueryParams && err == nil {
		err = decodeQueryParams(values, v)
	}

	return
}

func decode(r *http.Request, qp QueryParamsOption, maxMemory int64, v interface{}) (err error) {
	typ := r.Header.Get(contentTypeHeader)
	if idx := strings.Index(typ, ";"); idx != -1 {
		typ = typ[:idx]
	}

	switch typ {
	case nakedApplicationJSON:
		err = DecodeJSON(r, qp, maxMemory, v)
	case nakedApplicationXML:
		err = DecodeXML(r, qp, maxMemory, v)
	case applicationForm:
		err = DecodeForm(r, qp, v)
	case multipartForm:
		err = DecodeMultipartForm(r, qp, maxMemory, v)
	default:
		if qp == httpQueryParams {
			err = DecodeQueryParams(r, qp, v)
		}
	}

	return
}
