package validate

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// ---- fixtures covering diverse field shapes ----

type rvSub struct {
	Name string
	Age  int
}

type rvForm struct {
	// plain non-zero
	Name string
	Age  int
	// zero values
	Empty string
	Zero  int
	Off   bool
	// non-nil pointer (points to a zero value -> #217 leaf-pointer case)
	PtrFalse *bool
	// nil pointer
	PtrNil *int
	// nested struct
	Sub rvSub
	// slice / map
	Tags []string
	Meta map[string]string
	// named type
	Status rvStatus
	// unexported (non-interfaceable when accessed via reflect on an addressable value)
	secret string
}

type rvStatus int

// assertTryGetEquiv asserts TryGet and tryGetRV agree, plus matches the
// expected (val, exist, zero) literal triple.
func assertTryGetEquiv(t *testing.T, d *StructData, field string, wantVal any, wantExist, wantZero bool) {
	t.Helper()

	val, exist, zero := d.TryGet(field)
	assert.Eq(t, wantExist, exist, "TryGet exist mismatch for %q", field)
	assert.Eq(t, wantZero, zero, "TryGet zero mismatch for %q", field)
	if wantExist {
		assert.Eq(t, wantVal, val, "TryGet val mismatch for %q", field)
	} else {
		assert.Nil(t, val, "TryGet val should be nil when not exist for %q", field)
	}

	rv, e2, z2 := d.tryGetRV(field)
	assert.Eq(t, exist, e2, "tryGetRV exist must equal TryGet exist for %q", field)
	assert.Eq(t, zero, z2, "tryGetRV zero must equal TryGet zero for %q", field)
	if exist {
		assert.True(t, rv.IsValid(), "tryGetRV must return a valid value when exist for %q", field)
		assert.Eq(t, val, rv.Interface(), "tryGetRV.Interface() must deep-equal TryGet val for %q", field)
	}
}

func TestTryGetRV_Equivalence(t *testing.T) {
	fv := false
	form := &rvForm{
		Name:     "inhere",
		Age:      18,
		PtrFalse: &fv,
		PtrNil:   nil,
		Sub:      rvSub{Name: "sub", Age: 0},
		Tags:     []string{"a", "b"},
		Meta:     map[string]string{"k": "v"},
		Status:   rvStatus(0),
		secret:   "hidden",
	}

	d, err := FromStruct(form)
	assert.NoErr(t, err)

	t.Run("plain non-zero string", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Name", "inhere", true, false)
	})
	t.Run("plain non-zero int", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Age", 18, true, false)
	})
	t.Run("zero string", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Empty", "", true, true)
	})
	t.Run("zero int", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Zero", 0, true, true)
	})
	t.Run("bool false (zero, still exists)", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Off", false, true, true)
	})
	t.Run("named type zero", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Status", rvStatus(0), true, true)
	})
	t.Run("non-nil pointer to false (#217 top-level)", func(t *testing.T) {
		// top-level non-nil pointer is kept as-is; *bool->false is not nil, not zero-ptr
		assertTryGetEquiv(t, d, "PtrFalse", &fv, true, false)
	})
	t.Run("nil pointer => not exist", func(t *testing.T) {
		assertTryGetEquiv(t, d, "PtrNil", nil, false, false)
	})
	t.Run("slice field", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Tags", []string{"a", "b"}, true, false)
	})
	t.Run("map field", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Meta", map[string]string{"k": "v"}, true, false)
	})
	t.Run("missing top-level field => not exist", func(t *testing.T) {
		assertTryGetEquiv(t, d, "NoSuchField", nil, false, false)
	})

	// nested struct sub-path
	t.Run("nested non-zero", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Sub.Name", "sub", true, false)
	})
	t.Run("nested zero", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Sub.Age", 0, true, true)
	})
	t.Run("nested missing leaf => not exist", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Sub.NoSuch", nil, false, false)
	})
	t.Run("sub-path on non-container top => not exist", func(t *testing.T) {
		// Name is a string, not struct/array/slice/map
		assertTryGetEquiv(t, d, "Name.x", nil, false, false)
	})

	// wildcard early-return
	t.Run("wildcard returns whole sub-value", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Sub.*", form.Sub, true, false)
	})
	t.Run("wildcard on slice", func(t *testing.T) {
		assertTryGetEquiv(t, d, "Tags.*", []string{"a", "b"}, true, false)
	})

	// unexported / !CanInterface: accessed via FieldByName on addressable value
	t.Run("unexported field => not exist (!CanInterface)", func(t *testing.T) {
		// register the original casing so UpperFirst does not rewrite "secret"
		d.fieldNames["secret"] = 0
		// "secret" is a valid field but fv.CanInterface() is false -> exist=false.
		// Note: tryGetRV falls through to the bare `return`, so the named return fv
		// still holds the (valid but non-interfaceable) value — only exist=false is
		// the contract. This mirrors TryGet, which then returns (nil,false,false).
		_, exist, zero := d.tryGetRV("secret")
		assert.False(t, exist, "unexported field must be exist=false")
		assert.False(t, zero)

		val, exist2, zero2 := d.TryGet("secret")
		assert.False(t, exist2)
		assert.False(t, zero2)
		assert.Nil(t, val)
	})
}

// TestTryGetRV_CacheHit covers the d.fieldValues cache branch populated by Set().
func TestTryGetRV_CacheHit(t *testing.T) {
	form := &rvForm{Name: "inhere"}
	d, err := FromStruct(form)
	assert.NoErr(t, err)

	// Set() writes into d.fieldValues cache, so a later read goes through the
	// cache-hit branch in tryGetRV.
	_, err = d.Set("Name", "changed")
	assert.NoErr(t, err)

	val, exist, zero := d.TryGet("Name")
	assert.True(t, exist)
	assert.False(t, zero)
	assert.Eq(t, "changed", val)

	rv, e2, z2 := d.tryGetRV("Name")
	assert.True(t, e2)
	assert.False(t, z2)
	assert.Eq(t, "changed", rv.Interface())
}
