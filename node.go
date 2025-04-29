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

func (n *node) insertChild(numParams uint8, existing existingParams, path string, fullPath string, handler http.HandlerFunc) {
	var offset int // already handled bytes of the path
	// find prefix until first wildcard
	// (beginning with paramByte' or wildByte')
	for i, max := 0, len(path); numParams > 0; i++ {
		c := path[i]
		if c != paramByte && c != wildByte {
			continue
		}

		// find wildcard end
		// (either '/' or path end)
		end := i + 1
		for end < max && path[end] != slashByte {
			switch path[end] {
			// wildcard name must not contain ':' and '*'
			case paramByte, wildByte:
				panic("only one wildcard per path segment is allowed, has: '" + path[i:] + "' in path '" + fullPath + "'")
			default:
				end++
			}
		}

		// check if this node existing children,
		// which will be unreachable if a wildcard is inserted here
		if len(n.children) > 0 {
			panic("wildcard route '" + path[i:end] + "' conflicts with existing children in path '" + fullPath + "'")
		}

		if c == paramByte { // param
			// check if the wildcard has a name
			if end-i < 2 {
				panic("wildcards must be named with a non-empty name in path '" + fullPath + "'")
			}

			// split path at the beginning of the wildcard
			if i > 0 {
				n.path = path[offset:i]
				offset = i
			}

			child := &node{
				nType: hasParams,
			}
			n.children = []*node{child}
			n.wildChild = true
			n = child
			n.priority++
			numParams--
			// if the path doesn't end with the wildcard,
			// then there will be another non-wildcard subpath starting with '/'
			if end < max {
				existing.check(path[offset:end], fullPath)
				n.path = path[offset:end]
				offset = end
				child := &node{
					priority: 1,
				}
				n.children = []*node{child}
				n = child
			}
		} else { // catchAll
			if end != max || numParams > 1 {
				panic("character after the * symbol is not permitted, path '" + fullPath + "'")
			}

			if len(n.path) > 0 && n.path[len(n.path)-1] == '/' {
				panic("catch-all conflicts with existing handle for the path segment root in path '" + fullPath + "'")
			}

			// currently fixed width 1 for '/'
			i--
			if path[i] != slashByte {
				panic("no / before catch-all in path '" + fullPath + "'")
			}

			n.path = path[offset:i]
			// first node: catchAll node with empty path
			child := &node{
				wildChild: true,
				nType:     matchesAny,
			}
			n.children = []*node{child}
			n.indices = string(path[i])
			n = child
			n.priority++
			// second node: node holding the variable
			child = &node{
				path:     path[i:],
				nType:    matchesAny,
				handler:  handler,
				priority: 1,
			}
			n.children = []*node{child}
			return
		}
	}

	if n.nType == hasParams {
		existing.check(path[offset:], fullPath)
	}

	// insert remaining path part and handle to the leaf
	n.path = path[offset:]
	n.handler = handler
}

// incrementChildPriority increments priority of the given child and reorders if necessary.
func (n *node) incrementChildPriority(pos int) int {
	n.children[pos].priority++
	prio := n.children[pos].priority
	// adjust position (move to front)
	newPos := pos
	for newPos > 0 && n.children[newPos-1].priority < prio {
		// swap node positions
		n.children[newPos-1], n.children[newPos] = n.children[newPos], n.children[newPos-1]
		newPos--
	}

	// build new index char string
	if newPos != pos {
		n.indices = n.indices[:newPos] + // unchanged prefix, might be empty
			n.indices[pos:pos+1] + // the index char we move
			n.indices[newPos:pos] + n.indices[pos+1:] // rest without char at 'pos'
	}

	return newPos
}

func countParams(path string) (n uint8) {
	for i := 0; i < len(path) && n < 255; i++ {
		if path[i] == paramByte || path[i] == wildByte {
			n++
		}
	}

	if n >= 255 {
		panic("too many parameters defined in path, max is 255")
	}

	return uint8(n)
}
