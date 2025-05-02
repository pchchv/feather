package feather

import "net/http"

// RequestVars returns the request scoped variables tracked by feather.
func RequestVars(r *http.Request) ReqVars {
	rv := r.Context().Value(defaultContextIdentifier)
	if rv == nil {
		return new(requestVars)
	}

	return rv.(*requestVars)
}
