package feather

import "net/http"

// Middleware is pure's middleware definition.
type Middleware func(h http.HandlerFunc) http.HandlerFunc
