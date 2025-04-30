package feather

// ReqVars is the interface of request scoped variables tracked by feather.
type ReqVars interface {
	URLParam(pname string) string
}
