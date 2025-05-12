package feather

import (
	"context"
	"net/http"
	"strings"
	"sync"
)

const (
	WildcardParam            = "*wildcard"
	allowHeader              = "Allow"
	acceptEncodingHeader     = "Accept-Encoding"
	acceptedLanguageHeader   = "Accept-Language"
	contentTypeHeader        = "Content-Type"
	contentDispositionHeader = "Content-Disposition"
	xRealIPHeader            = "X-Real-Ip"
	xForwardedForHeader      = "X-Forwarded-For"
	varyHeader               = "Vary"
	slashByte                = '/'
	paramByte                = ':'
	basePath                 = "/"
	wildByte                 = '*'
	blank                    = ""
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
		name: "feather",
	}
)

// Middleware is feather's middleware definition.
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

// New Creates and returns a new feather instance.
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
	p.routeGroup.feather = p
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

// Serve returns an http.Handler to be used.
func (p *Mux) Serve() http.Handler {
	// is reserved for any logic that must occur before service begins,
	// i.e. although this router does not use priority to determine route order,
	// it is possible to add tree node sorting here
	return http.HandlerFunc(p.serveHTTP)
}

// SetRedirectTrailingSlash tells feather whether to attempt to fix the URL by trying to find it.
// lowercase -> with or without slash -> 404
func (p *Mux) SetRedirectTrailingSlash(set bool) {
	p.redirectTrailingSlash = set
}

// Register404 allows to override the handler function for routes not found.
// Runs after a route is not found, even after redirecting with the trailing slash.
func (p *Mux) Register404(notFound http.HandlerFunc, middleware ...Middleware) {
	h := notFound
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	p.http404 = h
}

// RegisterAutomaticOPTIONS specifies feather whether OPTION requests should be handled automatically.
// Manually configured OPTION handlers take precedence.
// By default, true.
func (p *Mux) RegisterAutomaticOPTIONS(middleware ...Middleware) {
	p.automaticallyHandleOPTIONS = true
	h := automaticOPTIONSHandler
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	p.httpOPTIONS = h
}

// RegisterMethodNotAllowed indicates feather whether the http 405 Method Not Allowed status code should be processed.
func (p *Mux) RegisterMethodNotAllowed(middleware ...Middleware) {
	p.handleMethodNotAllowed = true
	h := methodNotAllowedHandler
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}

	p.http405 = h
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

// serveHTTP conforms to the http.Handler interface.
func (p *Mux) serveHTTP(w http.ResponseWriter, r *http.Request) {
	var rv *requestVars
	var h http.HandlerFunc
	tree := p.trees[r.Method]
	if tree != nil {
		if h, rv = tree.find(r.URL.Path, p); h == nil {
			if p.redirectTrailingSlash && len(r.URL.Path) > 1 { // find again all lowercase
				orig := r.URL.Path
				lc := strings.ToLower(orig)
				if lc != r.URL.Path {
					if h, _ = tree.find(lc, p); h != nil {
						r.URL.Path = lc
						h = p.redirect(r.Method, r.URL.String())
						r.URL.Path = orig
						goto END
					}
				}

				if lc[len(lc)-1:] == basePath {
					lc = lc[:len(lc)-1]
				} else {
					lc = lc + basePath
				}

				if h, _ = tree.find(lc, p); h != nil {
					r.URL.Path = lc
					h = p.redirect(r.Method, r.URL.String())
					r.URL.Path = orig
					goto END
				}
			}
		} else {
			goto END
		}
	}

	if p.automaticallyHandleOPTIONS && r.Method == http.MethodOptions {
		if r.URL.Path == "*" { // check server-wide OPTIONS
			for m := range p.trees {
				if m != http.MethodOptions {
					w.Header().Add(allowHeader, m)
				}
			}
		} else {
			for m, ctree := range p.trees {
				if m == r.Method || m == http.MethodOptions {
					continue
				}

				if h, _ = ctree.find(r.URL.Path, p); h != nil {
					w.Header().Add(allowHeader, m)
				}
			}
		}

		w.Header().Add(allowHeader, http.MethodOptions)
		h = p.httpOPTIONS
		goto END
	}

	if p.handleMethodNotAllowed {
		var found bool
		for m, ctree := range p.trees {
			if m != r.Method {
				if h, _ = ctree.find(r.URL.Path, p); h != nil {
					w.Header().Add(allowHeader, m)
					found = true
				}
			}
		}

		if found {
			h = p.http405
			goto END
		}
	}

	// not found
	h = p.http404

END:
	if rv != nil {
		rv.formParsed = false
		// store on context
		r = r.WithContext(rv.ctx)
	}

	h(w, r)

	if rv != nil {
		p.pool.Put(rv)
	}
}
