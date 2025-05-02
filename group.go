package feather

import (
	"net/http"
	"strconv"
	"strings"
)

// routeGroup containing all fields and methods for use.
type routeGroup struct {
	prefix     string
	middleware []Middleware
	feather    *Mux
}

// Get adds a GET route & handler to the router.
func (g *routeGroup) Get(path string, h http.HandlerFunc) {
	g.handle(http.MethodGet, path, h)
}

// Delete adds a DELETE route & handler to the router.
func (g *routeGroup) Delete(path string, h http.HandlerFunc) {
	g.handle(http.MethodDelete, path, h)
}

// Post adds a POST route & handler to the router.
func (g *routeGroup) Post(path string, h http.HandlerFunc) {
	g.handle(http.MethodPost, path, h)
}

// Put adds a PUT route & handler to the router.
func (g *routeGroup) Put(path string, h http.HandlerFunc) {
	g.handle(http.MethodPut, path, h)
}

// Patch adds a PATCH route & handler to the router.
func (g *routeGroup) Patch(path string, h http.HandlerFunc) {
	g.handle(http.MethodPatch, path, h)
}

// Options adds an OPTIONS route & handler to the router.
func (g *routeGroup) Options(path string, h http.HandlerFunc) {
	g.handle(http.MethodOptions, path, h)
}

// Use adds a middleware handler to the group middleware chain.
func (g *routeGroup) Use(m ...Middleware) {
	g.middleware = append(g.middleware, m...)
}

// Trace adds a TRACE route & handler to the router.
func (g *routeGroup) Trace(path string, h http.HandlerFunc) {
	g.handle(http.MethodTrace, path, h)
}

func (g *routeGroup) handle(method string, path string, handler http.HandlerFunc) {
	if i := strings.Index(path, "//"); i != -1 {
		panic("Bad path '" + path + "' contains duplicate // at index:" + strconv.Itoa(i))
	}

	h := handler
	for i := len(g.middleware) - 1; i >= 0; i-- {
		h = g.middleware[i](h)
	}

	tree := g.feather.trees[method]
	if tree == nil {
		tree = new(node)
		g.feather.trees[method] = tree
	}

	pCount := tree.addRoute(g.prefix+path, h) + 1
	if pCount > g.feather.mostParams {
		g.feather.mostParams = pCount
	}
}
