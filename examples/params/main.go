package main

import (
	"net/http"

	"github.com/pchchv/feather"
	lr "github.com/pchchv/feather/examples/middleware/logging_recovery"
)

func main() {
	p := feather.New()
	p.Use(lr.LoggingAndRecovery(true))
	p.Get("/user/:id", user)

	http.ListenAndServe(":3007", p.Serve())
}

func user(w http.ResponseWriter, r *http.Request) {
	rv := feather.RequestVars(r)
	w.Write([]byte("USER_ID:" + rv.URLParam("id")))
}
