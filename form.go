package feather

import (
	"net/url"

	"github.com/pchchv/form"
)

// FormEncoder is the type used for encoding form data.
type FormEncoder interface {
	Encode(interface{}) (url.Values, error)
}

// DefaultFormEncoder of this package, which is configurable.
var DefaultFormEncoder FormEncoder = form.NewEncoder()
