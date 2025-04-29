package feather

const (
	isRoot nodeType = iota + 1
	hasParams
	matchesAny
)

type nodeType uint8

type existingParams map[string]struct{}
