package validate

import (
	"reflect"
	"testing"

	"github.com/gookit/goutil/x/assert"

	"github.com/gookit/validate/v2/internal/fieldval"
)

func ptrOf[T any](v T) *T { return &v }

// TestFieldValue_rV covers rV() lazy construction and the nilRVal substitution
// for untyped-nil src (matching callValidatorValue's #125 fix).
func TestFieldValue_rV(t *testing.T) {
	t.Run("normal value", func(t *testing.T) {
		fv := fieldval.New("n", "hi")
		rv := fv.RV()
		assert.True(t, rv.IsValid())
		assert.Eq(t, reflect.String, rv.Kind())
		assert.Eq(t, "hi", rv.String())
		// cached: second call returns same kind
		assert.Eq(t, rv.Kind(), fv.RV().Kind())
	})

	t.Run("untyped nil -> nilRVal", func(t *testing.T) {
		fv := fieldval.New("n", nil)
		rv := fv.RV()
		// nilRVal is reflect.ValueOf(NilObject{}): valid, struct kind
		assert.True(t, rv.IsValid())
		assert.Eq(t, reflect.Struct, rv.Kind())
		assert.True(t, IsNilObj(rv.Interface()))
	})

	t.Run("typed nil pointer stays valid ptr", func(t *testing.T) {
		var p *int
		fv := fieldval.New("n", p)
		rv := fv.RV()
		assert.True(t, rv.IsValid())
		assert.Eq(t, reflect.Ptr, rv.Kind())
		assert.True(t, rv.IsNil())
	})

	t.Run("rT returns type", func(t *testing.T) {
		fv := fieldval.New("n", 42)
		assert.Eq(t, reflect.TypeOf(42), fv.RT())
	})
}

// TestFieldValue_realV covers single-level non-nil pointer deref, matching the
// previous inline handling in callValidatorValue.
func TestFieldValue_realV(t *testing.T) {
	t.Run("non-ptr value unchanged", func(t *testing.T) {
		fv := fieldval.New("n", 7)
		assert.Eq(t, reflect.Int, fv.RealV().Kind())
		assert.Eq(t, int64(7), fv.RealV().Int())
	})

	t.Run("ptr to valid value is dereferenced", func(t *testing.T) {
		fv := fieldval.New("n", ptrOf(123))
		rv := fv.RealV()
		assert.Eq(t, reflect.Int, rv.Kind())
		assert.Eq(t, int64(123), rv.Int())
	})

	t.Run("nil ptr kept as ptr", func(t *testing.T) {
		var p *int
		fv := fieldval.New("n", p)
		rv := fv.RealV()
		assert.Eq(t, reflect.Ptr, rv.Kind())
		assert.True(t, rv.IsNil())
	})

	t.Run("double pointer single-level deref only", func(t *testing.T) {
		inner := ptrOf(9)
		fv := fieldval.New("n", &inner) // **int
		rv := fv.RealV()
		// only one level removed -> still *int
		assert.Eq(t, reflect.Ptr, rv.Kind())
		assert.Eq(t, reflect.Int, rv.Elem().Kind())
	})
}

// TestFieldValue_isZero aligns isZero() with reflect.Value.IsZero() which is
// what StructData.TryGet uses for the returned zero flag.
func TestFieldValue_isZero(t *testing.T) {
	cases := []struct {
		name string
		src  any
		want bool
	}{
		{"non-zero string", "x", false},
		{"empty string", "", true},
		{"zero int", 0, true},
		{"non-zero int", 5, false},
		{"false bool", false, true},
		{"true bool", true, false},
		{"nil slice", []int(nil), true},
		{"empty slice", []int{}, false}, // empty-but-non-nil slice is NOT zero
		{"non-empty slice", []int{1}, false},
		{"nil map", map[string]int(nil), true},
		{"empty map", map[string]int{}, false}, // non-nil map is NOT zero
		{"nil ptr", (*int)(nil), true},
		{"ptr to zero value", ptrOf(0), false}, // pointer itself is non-nil -> not zero
		{"ptr to value", ptrOf(3), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fv := fieldval.New("n", c.src)
			assert.Eq(t, c.want, fv.IsZero())
			// cached: second call consistent
			assert.Eq(t, c.want, fv.IsZero())
		})
	}

	t.Run("untyped nil isZero", func(t *testing.T) {
		// rV() -> nilRVal (NilObject{} zero value) -> IsZero == true
		fv := fieldval.New("n", nil)
		assert.True(t, fv.IsZero())
	})
}

// TestFieldValue_isEmpty must give identical results to the public IsEmpty(any)
// for the same inputs.
func TestFieldValue_isEmpty(t *testing.T) {
	cases := []struct {
		name string
		src  any
	}{
		{"non-empty string", "abc"},
		{"empty string", ""},
		{"zero int", 0},
		{"non-zero int", 9},
		{"false bool", false},
		{"true bool", true},
		{"nil slice", []int(nil)},
		{"empty slice", []int{}},
		{"non-empty slice", []int{1, 2}},
		{"nil map", map[string]int(nil)},
		{"empty map", map[string]int{}},
		{"non-empty map", map[string]int{"a": 1}},
		{"untyped nil", nil},
		{"typed nil ptr", (*int)(nil)},
		{"ptr to zero value", ptrOf(0)},
		{"ptr to value", ptrOf(5)},
		{"float zero", 0.0},
		{"struct value", struct{ A int }{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fv := fieldval.New("n", c.src)
			want := IsEmpty(c.src)
			assert.Eq(t, want, fv.IsEmpty(), "isEmpty must match IsEmpty(any)")
			// cached: second call consistent
			assert.Eq(t, want, fv.IsEmpty())
		})
	}
}

// TestNewFieldValueRV covers the reflect.Value-based constructor reserved for P2.
func TestNewFieldValueRV(t *testing.T) {
	t.Run("from valid reflect.Value", func(t *testing.T) {
		fv := fieldval.NewRV("n", reflect.ValueOf("hello"))
		assert.Eq(t, reflect.String, fv.RV().Kind())
		assert.Eq(t, "hello", fv.Src)
		assert.False(t, fv.IsEmpty())
		assert.False(t, fv.IsZero())
	})

	t.Run("from invalid reflect.Value -> nilRVal", func(t *testing.T) {
		fv := fieldval.NewRV("n", reflect.Value{})
		rv := fv.RV()
		assert.True(t, rv.IsValid())
		assert.True(t, IsNilObj(rv.Interface()))
	})

	t.Run("equivalent to fieldval.New for same value", func(t *testing.T) {
		a := fieldval.New("n", 42)
		b := fieldval.NewRV("n", reflect.ValueOf(42))
		assert.Eq(t, a.RV().Kind(), b.RV().Kind())
		assert.Eq(t, a.IsEmpty(), b.IsEmpty())
		assert.Eq(t, a.IsZero(), b.IsZero())
	})
}
