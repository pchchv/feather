package main

import (
	"net/http"

	"github.com/pchchv/feather"
	lr "github.com/pchchv/feather/examples/middleware/logging-recovery"
)

func main() {
	p := feather.New()
	p.Use(lr.LoggingAndRecovery(false))
	p.Get("/", helloWorld)
	http.ListenAndServe(":3007", p.Serve())
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}
