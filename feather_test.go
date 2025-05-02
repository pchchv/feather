package feather

import "net/http"

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
