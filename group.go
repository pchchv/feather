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
