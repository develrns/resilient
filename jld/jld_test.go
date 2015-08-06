package jld

import (
	"testing"
)

func TestNewV(test *testing.T) {
	var (
		t        TypeID
		v, vt    interface{}
		tok, vok bool
		valobj   map[string]interface{}
	)

	t = NewTypeID("type", "")
	v = "value"
	valobj = NewV(t, v)
	_, tok = valobj["@type"].(string)
	vt, vok = valobj["@value"].(string)
	switch {
	case !tok:
		test.Errorf("NewV type: %v value: %v", t, v)
	case !vok:
		test.Errorf("NewV type: %v value: %v", t, v)
	}

	v = 1
	valobj = NewV(t, v)
	_, tok = valobj["@type"].(string)
	vt, vok = valobj["@value"].(int)
	switch {
	case !tok:
		test.Errorf("NewV type: %v value: %v", t, v)
	case !vok:
		test.Errorf("NewV type: %v value: %v", t, v)
	}

	v = float32(1.0)
	valobj = NewV(t, v)
	_, tok = valobj["@type"].(string)
	vt, vok = valobj["@value"].(float32)
	switch {
	case !tok:
		test.Errorf("NewV type: %v value: %v", t, v)
	case !vok:
		test.Errorf("NewV type: %v value: %v", t, v)
	}

	v = float64(1.0)
	valobj = NewV(t, v)
	_, tok = valobj["@type"].(string)
	vt, vok = valobj["@value"].(float64)
	switch {
	case !tok:
		test.Errorf("NewV type: %v value: %v", t, v)
	case !vok:
		test.Errorf("NewV type: %v value: %v", t, v)
	}

	v = true
	valobj = NewV(t, v)
	_, tok = valobj["@type"].(string)
	vt, vok = valobj["@value"].(bool)
	switch {
	case !tok:
		test.Errorf("NewV type: %v value: %v", t, v)
	case !vok:
		test.Errorf("NewV type: %v value: %v", t, v)
	}

	v = nil
	valobj = NewV(t, v)
	_, tok = valobj["@type"].(string)
	vt = valobj["@value"]
	switch {
	case !tok:
		test.Errorf("NewV type: %v value: %v", t, v)
	case vt != nil:
		test.Errorf("NewV type: %v value: %v", t, v)
	}

	v = []int{1, 2}
	valobj = NewV(t, v)
	_, tok = valobj["@type"].(string)
	vt = valobj["@value"]
	switch {
	case !tok:
		test.Errorf("NewV type: %v value: %v", t, v)
	case vt != nil:
		test.Errorf("NewV type: %v value: %v", t, v)
	}

}

func TestNewN(test *testing.T) {
}
