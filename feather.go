package feather

import (
	"context"
	"net/http"
	"sync"
)

const (
	WildcardParam = "*wildcard"
	slashByte     = '/'
	paramByte     = ':'
	basePath      = "/"
	wildByte      = '*'
	blank         = ""
)

var (
	default404Handler = func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	}

	methodNotAllowedHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}

	automaticOPTIONSHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	defaultContextIdentifier = &struct {
		name string
	}{
		name: "pure",
	}
)

// Middleware is pure's middleware definition.
type Middleware func(h http.HandlerFunc) http.HandlerFunc

// Mux is the main request multiplexer.
type Mux struct {
	routeGroup
	trees       map[string]*node
	pool        sync.Pool        // pool is used for reusable request scoped RequestVars content
	http404     http.HandlerFunc // 404 Not Found
	http405     http.HandlerFunc // 405 Method Not Allowed
	httpOPTIONS http.HandlerFunc
	mostParams  uint8 // mostParams used to keep track of the most amount of params in any URL and this will set the default capacity of each Params
	// redirectTrailingSlash enables automatic redirection if
	// the current route can't be matched but a handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo,
	// the client is redirected to /foo with http status code 301 for GET requests and 307 for all other request methods.
	redirectTrailingSlash bool
	// If enabled, the router checks if another method is allowed for the current route,
	// if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed' and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound handler.
	handleMethodNotAllowed bool
	// If enabled automatically handles OPTION requests; manually configured OPTION
	// handlers take presidence. default true
	automaticallyHandleOPTIONS bool
}

// New Creates and returns a new Pure instance.
func New() *Mux {
	p := &Mux{
		routeGroup: routeGroup{
			middleware: make([]Middleware, 0),
		},
		trees:                      make(map[string]*node),
		mostParams:                 0,
		http404:                    default404Handler,
		http405:                    methodNotAllowedHandler,
		httpOPTIONS:                automaticOPTIONSHandler,
		redirectTrailingSlash:      true,
		handleMethodNotAllowed:     false,
		automaticallyHandleOPTIONS: false,
	}
	p.routeGroup.pure = p
	p.pool.New = func() interface{} {
		rv := &requestVars{
			params: make(urlParams, p.mostParams),
		}
		rv.ctx = context.WithValue(context.Background(), defaultContextIdentifier, rv)
		return rv
	}

	return p
}

type urlParam struct {
	key   string
	value string
}

type urlParams []urlParam

// Get returns the URL parameter for the given key, or blank if not found.
func (p urlParams) Get(key string) (param string) {
	for i := 0; i < len(p); i++ {
		if p[i].key == key {
			param = p[i].value
			break
		}
	}

	return
}

func (p *Mux) redirect(method string, to string) (h http.HandlerFunc) {
	code := http.StatusMovedPermanently
	if method != http.MethodGet {
		code = http.StatusPermanentRedirect
	}

	h = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, to, code)
	}

	for i := len(p.middleware) - 1; i >= 0; i-- {
		h = p.middleware[i](h)
	}

	return
}
