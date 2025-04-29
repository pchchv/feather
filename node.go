package feather

import (
	"net/http"
	"net/url"
)

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

// addRoute adds the node with the given handle to the path.
// Middleware is set here because it needs to transfer all route's middlewares
// (it is a chain of functions) with its handler to the node.
func (n *node) addRoute(path string, handler http.HandlerFunc) (lp uint8) {
	var err error
	if path == blank {
		path = basePath
	}

	existing := make(existingParams)
	fullPath := path
	if path, err = url.QueryUnescape(path); err != nil {
		panic("Qery Unescape Error on path '" + fullPath + "': " + err.Error())
	}

	fullPath = path
	n.priority++
	numParams := countParams(path)
	lp = numParams
	// non-empty tree
	if len(n.path) > 0 || len(n.children) > 0 {
	walk:
		for {
			// find the longest common prefix.
			// this also implies that the common prefix contains no : or *
			// since the existing key can't contain those chars.
			var i int
			max := min(len(path), len(n.path))
			for i < max && path[i] == n.path[i] {
				i++
			}

			// split edge
			if i < len(n.path) {
				child := node{
					path:      n.path[i:],
					wildChild: n.wildChild,
					indices:   n.indices,
					children:  n.children,
					handler:   n.handler,
					priority:  n.priority - 1,
				}
				n.children = []*node{&child}
				// []byte for proper unicode char conversion
				n.indices = string([]byte{n.path[i]})
				n.path = path[:i]
				n.handler = nil
				n.wildChild = false
			}

			// make new node a child of this node
			if i < len(path) {
				path = path[i:]
				if n.wildChild {
					n = n.children[0]
					n.priority++
					numParams--
					existing.check(n.path, fullPath)
					// check if the wildcard matches
					if len(path) >= len(n.path) && n.path == path[:len(n.path)] {
						// check for longer wildcard, e.g. :name and :names
						if len(n.path) >= len(path) || path[len(n.path)] == slashByte {
							continue walk
						}
					}

					panic("path segment '" + path + "' conflicts with existing wildcard '" + n.path + "' in path '" + fullPath + "'")
				}

				c := path[0]
				// slash after param
				if n.nType == hasParams && c == slashByte && len(n.children) == 1 {
					n = n.children[0]
					n.priority++
					continue walk
				}

				// check if a child with the next path byte exists
				for i := 0; i < len(n.indices); i++ {
					if c == n.indices[i] {
						i = n.incrementChildPriority(i)
						n = n.children[i]
						continue walk
					}
				}

				// otherwise insert it
				if c != paramByte && c != wildByte {

					// []byte for proper unicode char conversion
					n.indices += string([]byte{c})
					child := &node{}
					n.children = append(n.children, child)
					n.incrementChildPriority(len(n.indices) - 1)
					n = child
				}

				n.insertChild(numParams, existing, path, fullPath, handler)
				return
			} else if i == len(path) { // make node a (in-path) leaf
				if n.handler != nil {
					panic("handlers are already registered for path '" + fullPath + "'")
				}
				n.handler = handler
			}

			return
		}
	} else { // empty tree
		n.insertChild(numParams, existing, path, fullPath, handler)
		n.nType = isRoot
	}

	return
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
