package feather

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMatch(t *testing.T) {
	p := New()
	p.Match([]string{http.MethodConnect, http.MethodDelete, http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace}, "/test", defaultHandler)
	hf := p.Serve()
	tests := []struct {
		method string
	}{
		{
			method: http.MethodConnect,
		},
		{
			method: http.MethodDelete,
		},
		{
			method: http.MethodGet,
		},
		{
			method: http.MethodHead,
		},
		{
			method: http.MethodOptions,
		},
		{
			method: http.MethodPatch,
		},
		{
			method: http.MethodPost,
		},
		{
			method: http.MethodPut,
		},
		{
			method: http.MethodTrace,
		},
	}

	for _, tt := range tests {
		req, err := http.NewRequest(tt.method, "/test", nil)
		if err != nil {
			t.Errorf("Expected 'nil' Got '%s'", err)
		}

		res := httptest.NewRecorder()
		hf.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Errorf("Expected '%d' Got '%d'", http.StatusOK, res.Code)
		}

		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Errorf("Expected 'nil' Got '%s'", err)
		}

		s := string(b)
		if s != tt.method {
			t.Errorf("Expected '%s' Got '%s'", tt.method, s)
		}
	}
}
