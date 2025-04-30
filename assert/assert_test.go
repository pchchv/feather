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
