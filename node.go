package feather

import "net/http"

const (
	isRoot nodeType = iota + 1
	hasParams
	matchesAny
)

type nodeType uint8

type existingParams map[string]struct{}

func (e existingParams) check(param string, path string) {
	if _, ok := e[param]; ok {
		panic("duplicate param name '" + param + "' detected for route '" + path + "'")
	}

	e[param] = struct{}{}
}

type node struct {
	path      string
	indices   string
	children  []*node
	handler   http.HandlerFunc
	priority  uint32
	nType     nodeType
	wildChild bool
}
