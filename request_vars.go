package feather

import "context"

// ReqVars is the interface of request scoped variables tracked by feather.
type ReqVars interface {
	URLParam(pname string) string
}

type requestVars struct {
	ctx        context.Context // holds a copy of parent requestVars
	params     urlParams
	formParsed bool
}

// Params returns the current routes Params.
func (r *requestVars) URLParam(pname string) string {
	return r.params.Get(pname)
}
