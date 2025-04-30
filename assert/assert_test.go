package assert

import "testing"

func TestEquals(t *testing.T) {
	type Test struct {
		Name string
	}

	tst := &Test{
		Name: "joeybloggs",
	}

	Equal(t, tst, tst)
	NotEqual(t, tst, nil)
	NotEqual(t, nil, tst)

	type TestMap map[string]string

	var tm TestMap
	Equal(t, tm, nil)
	Equal(t, nil, tm)

	var iface interface{}
	var iface2 interface{}
	iface = 1
	Equal(t, iface, 1)
	NotEqual(t, iface, iface2)
}

func TestRegexMatchAndNotMatch(t *testing.T) {
	goodRegex := "^(.*/vendor/)?github.com/pchchv/assert$"
	MatchRegex(t, "github.com/pchchv/assert", goodRegex)
	MatchRegex(t, "/vendor/github.com/pchchv/assert", goodRegex)
	NotMatchRegex(t, "/vendor/github.com/pchchv/test", goodRegex)
}

func CustomErrorHandler(t testing.TB, errs map[string]string, key, expected string) {
	val, ok := errs[key]
	EqualSkip(t, 2, ok, true)
	NotEqualSkip(t, 2, val, nil)
	EqualSkip(t, 2, val, expected)
}
