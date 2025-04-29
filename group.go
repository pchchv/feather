package feather

// routeGroup containing all fields and methods for use.
type routeGroup struct {
	prefix     string
	middleware []Middleware
	pure       *Mux
}
