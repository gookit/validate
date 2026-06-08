package validators

import (
	"math"
	"reflect"
	"testing"

	"github.com/gookit/goutil/x/assert"

	"github.com/gookit/validate/v2/internal/reflectx"
)

// named types: must NOT take compareRV fast paths (goutil mathutil dispatches on
// the concrete type, so a named numeric/string falls to compareAny -> false).
type myInt int
type myUint uint
type myFloat64 float64
type myFloat32 float32
type myString string

// TestCompareRV_EquivCompareAny is the hard correctness gate for RFC R4.3: for
// every (v, dst, op) the box-free compareRV must return EXACTLY what the
// pre-refactor compareAny(v, dst, op) returns. Both are computed independently.
func TestCompareRV_EquivCompareAny(t *testing.T) {
	ops := []string{">", ">=", "<", "<=", "!=", "=="}

	srcVals := []any{
		// plain signed ints (incl. negative, zero, MaxInt64)
		int(5), int(-3), int(0), int8(5), int8(-3), int16(5), int32(5),
		int64(5), int64(math.MaxInt64), int64(math.MinInt64),
		// plain unsigned ints (incl. MaxUint64 -> int64 wrap to -1)
		uint(5), uint8(5), uint8(255), uint16(5), uint32(5),
		uint64(5), uint64(math.MaxUint64),
		// plain floats (incl. 1.5, integral 2.0, negative)
		float64(1.5), float64(2.0), float64(-3.5), float64(5),
		float32(1.5), float32(2.0), float32(-3.5), float32(5),
		// plain strings
		"5", "abc", "", "10",
		// named types -> must fall to compareAny fallback
		myInt(5), myInt(-3), myUint(5), myUint(255),
		myFloat64(1.5), myFloat64(5), myFloat32(1.5), myFloat32(5),
		myString("5"), myString("abc"),
		// bool + NilObject -> mathutil default -> ToInt64 error -> false
		true, false, reflectx.NilObject{},
	}

	dstVals := []any{
		int64(5), int64(-3), int64(0), int64(120),
		int(5), int(1), float64(5), float64(1.5), float64(120),
		"5", "abc", "x", "", "10",
		// incompatible / nil dst types (nil exercises mathutil.Compare's nil guard)
		[]int{1, 2}, struct{ X int }{1}, reflectx.NilObject{}, nil,
	}

	count := 0
	for _, v := range srcVals {
		rv := reflect.ValueOf(v)
		if !rv.IsValid() { // reflect.ValueOf(nil) — skip, never reaches compareRV
			continue
		}
		for _, dst := range dstVals {
			for _, op := range ops {
				want := compareAny(v, dst, op)
				got := compareRV(rv, dst, op)
				count++
				if got != want {
					t.Fatalf("compareRV mismatch: v=%#v (%T) dst=%#v (%T) op=%q => got=%v want=%v",
						v, v, dst, dst, op, got, want)
				}
			}
		}
	}
	t.Logf("compareRV equivalence cases checked: %d", count)
}

// TestCompareRV_NamedTypesFallToFalse pins the documented behavior that named
// numeric/string types do NOT take the fast path and yield mathutil's
// default-ToInt64-error result (false), identical to current production code.
func TestCompareRV_NamedTypesFallToFalse(t *testing.T) {
	cases := []struct {
		v any
	}{
		{myInt(5)}, {myUint(5)}, {myFloat64(5)}, {myFloat32(5)}, {myString("5")},
	}
	for _, c := range cases {
		rv := reflect.ValueOf(c.v)
		// dst=5, op=">=" — a plain int 5 >= 5 would be true, but a named type
		// cannot be converted by goutil's ToInt64 -> both paths must be false.
		assert.False(t, compareRV(rv, 5, ">="))
		assert.False(t, compareRV(rv, 5, "=="))
		assert.Eq(t, compareAny(c.v, 5, ">="), compareRV(rv, 5, ">="))
	}
}

// TestCompareRV_PlainFastPaths spot-checks the box-free fast paths return the
// expected logical results (sanity on top of the equivalence gate).
func TestCompareRV_PlainFastPaths(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assert.True(t, compareRV(reflect.ValueOf(30), 1, ">="))
		assert.True(t, compareRV(reflect.ValueOf(30), 120, "<="))
		assert.False(t, compareRV(reflect.ValueOf(0), 1, ">="))
	})
	t.Run("uint64-wrap", func(t *testing.T) {
		// int64(math.MaxUint64) == -1, so it is < 0
		assert.True(t, compareRV(reflect.ValueOf(uint64(math.MaxUint64)), 0, "<"))
		assert.Eq(t,
			compareAny(uint64(math.MaxUint64), 0, "<"),
			compareRV(reflect.ValueOf(uint64(math.MaxUint64)), 0, "<"))
	})
	t.Run("float", func(t *testing.T) {
		assert.True(t, compareRV(reflect.ValueOf(1.5), 1.0, ">"))
		assert.True(t, compareRV(reflect.ValueOf(float32(1.5)), 1.0, ">"))
	})
	t.Run("string", func(t *testing.T) {
		assert.True(t, compareRV(reflect.ValueOf("b"), "a", ">"))
		assert.True(t, compareRV(reflect.ValueOf("5"), 5, "=="))
	})
}
