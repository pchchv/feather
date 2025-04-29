package feather

import (
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
