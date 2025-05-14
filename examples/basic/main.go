package main

import (
	"net/http"

	"github.com/pchchv/feather"
)

func main() {
	p := feather.New()
	p.Get("/", helloWorld)
	http.ListenAndServe(":3007", p.Serve())
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello World"))
}
