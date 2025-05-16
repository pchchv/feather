# feather [![Godoc Reference](https://pkg.go.dev/badge/github.com/pchchv/feather)](https://pkg.go.dev/github.com/pchchv/feather)

Feather is a radix-tree based fast HTTP router that adheres to Go's native implementations of the `net/http` package, essentially keeping the implementation of feather handlers using the `context` package.

# Features

- adheres to native Go implementations, providing helper functions for convenience
- **fast and efficient** - feather uses custom version of the radix tree and is therefore incredibly fast and efficient