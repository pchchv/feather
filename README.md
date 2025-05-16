# feather [![Godoc Reference](https://pkg.go.dev/badge/github.com/pchchv/feather)](https://pkg.go.dev/github.com/pchchv/feather)

Feather is a radix-tree based fast HTTP router that adheres to Go's native implementations of the `net/http` package, essentially keeping the implementation of feather handlers using the `context` package.

# Features

- adheres to native Go implementations, providing helper functions for convenience
- **fast and efficient** - feather uses custom version of the radix tree and is therefore incredibly fast and efficient

# Installation
 
```sh
go get github.com/pchchv/form
```

# Usage

```go
package main

import (
	"net/http"

	"github.com/pchchv/feather"
	lr "github.com/pchchv/feather/examples/middleware/logging-recovery"
)

func main() {
	p := feather.New()
	p.Use(lr.LoggingAndRecovery(true))
	p.Get("/", helloWorld)
	http.ListenAndServe(":3007", p.Serve())
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}
```

## RequestVars

This is an interface that is used to pass variables and functions associated with a query using `context.Context`. It is implemented this way because getting values from `context` is not the fastest, and so using this the router can store multiple pieces of information, reducing the lookup time to a single stored `RequestVars`.

Only URL/SEO parameters are stored in `RequestVars`, but if other parameters are added, they can simply be added to `RequestVars` and no additional lookup time is required.

## URL Params

```go
p := p.New()
// the matching param will be stored in the context's params with name "id"
p.Get("/user/:id", UserHandler)
// extract params like so
rv := feather.RequestVars(r) // done this way so only have to extract from context once, read above
rv.URLParam(paramname)
// serve css, js etc.. feather.RequestVars(r).URLParam(feather.WildcardParam) will return the remaining path if 
// you need to use it in a custom handler...
p.Get("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))).ServeHTTP)
...
```

**Note:** Since this router has only explicit matches, you can not register static routes and parameters for the same path segment. For example you can not register the patterns /user/new and /user/:user for the same request method at the same time. The routing of different request methods is independent from each other. I was initially against this, however it nearly cost me in a large web application where the dynamic param value say :type actually could have matched another static route and that's just too dangerous and so it is not allowed.