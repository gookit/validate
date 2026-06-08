package fieldval

import (
	"reflect"
	"testing"

	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/goutil/x/assert"

	"github.com/gookit/validate/v2/internal/reflectx"
)

// refIsEmpty mirrors the public validate.IsEmpty(any) semantics byte-for-byte:
// untyped-nil -> true; string -> s==""; else reflects.IsEmpty(reflect.ValueOf).
// This is the reference oracle for FieldValue.IsEmpty() equivalence.
func refIsEmpty(val any) bool {
	if val == nil {
		return true
	}
	if s, ok := val.(string); ok {
		return s == ""
	}
	return reflects.IsEmpty(reflect.ValueOf(val))
}

// refValToString mirrors the pre-refactor validating.valToString(any) byte-for-byte:
// string -> (s,true); nil -> ("",false); Kind String -> (rv.String(),true);
// else -> ToString(val),(s,err==nil). Reference oracle for FieldValue.String().
func refValToString(val any) (string, bool) {
	if s, ok := val.(string); ok {
		return s, true
	}
	if val == nil {
		return "", false
	}
	if rv := reflect.ValueOf(val); rv.Kind() == reflect.String {
		return rv.String(), true
	}
	s, err := strutil.ToString(val)
	return s, err == nil
}

// myStr is a named string type, must surface as Kind String (box-free path).
type myStr string

type someStruct struct{ A int }

// equivCases returns the equivalence matrix values plus whether the value is an
// untyped nil (for which NewRV cannot be exercised — reflect.ValueOf(nil) is
// invalid; that path is covered by New only).
func equivCases() []struct {
	name      string
	val       any
	untypedNil bool
} {
	var nilPtr *int
	x := 7
	return []struct {
		name       string
		val        any
		untypedNil bool
	}{
		{"empty-string", "", false},
		{"nonempty-string", "x", false},
		{"named-empty-string", myStr(""), false},
		{"named-string", myStr("abc"), false},
		{"int-zero", 0, false},
		{"int-one", 1, false},
		{"bool-false", false, false},
		{"bool-true", true, false},
		{"float-zero", 0.0, false},
		{"float-nonzero", 1.5, false},
		{"empty-slice", []int{}, false},
		{"nonempty-slice", []int{1}, false},
		{"nil-slice", []int(nil), false},
		{"empty-map", map[string]int{}, false},
		{"nonempty-map", map[string]int{"a": 1}, false},
		{"ptr-nonnil", &x, false},
		{"ptr-nil", nilPtr, false},
		{"struct", someStruct{A: 1}, false},
		{"nil-object", reflectx.NilObject{}, false},
		{"untyped-nil", nil, true},
	}
}

func TestFieldValue_IsEmpty_Equivalence(t *testing.T) {
	n := 0
	for _, c := range equivCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			want := refIsEmpty(c.val)

			// New path (eager src)
			got := New("", c.val).IsEmpty()
			assert.Eq(t, want, got, "New IsEmpty mismatch for %s", c.name)
			n++

			// NewRV path (lazy src) — skip untyped nil (invalid reflect.Value)
			if !c.untypedNil {
				gotRV := NewRV("", reflect.ValueOf(c.val)).IsEmpty()
				assert.Eq(t, want, gotRV, "NewRV IsEmpty mismatch for %s", c.name)
				n++
			}
		})
	}
	t.Logf("IsEmpty equivalence assertions: %d", n)
}

func TestFieldValue_String_Equivalence(t *testing.T) {
	n := 0
	for _, c := range equivCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			wantS, wantOk := refValToString(c.val)

			gotS, gotOk := New("", c.val).String()
			assert.Eq(t, wantOk, gotOk, "New String ok mismatch for %s", c.name)
			assert.Eq(t, wantS, gotS, "New String value mismatch for %s", c.name)
			n++

			if !c.untypedNil {
				gotS2, gotOk2 := NewRV("", reflect.ValueOf(c.val)).String()
				assert.Eq(t, wantOk, gotOk2, "NewRV String ok mismatch for %s", c.name)
				assert.Eq(t, wantS, gotS2, "NewRV String value mismatch for %s", c.name)
				n++
			}
		})
	}
	t.Logf("String equivalence assertions: %d", n)
}

// TestFieldValue_Src_LazyEquivalence ensures NewRV lazily materializes src to the
// exact same value the pre-refactor eager `f.Src = rv.Interface()` produced.
func TestFieldValue_Src_LazyEquivalence(t *testing.T) {
	for _, c := range equivCases() {
		c := c
		if c.untypedNil {
			// New(nil).Src() == nil
			assert.Nil(t, New("", c.val).Src())
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			// NewRV materialized src must DeepEqual rv.Interface().
			got := NewRV("", reflect.ValueOf(c.val)).Src()
			assert.True(t, reflect.DeepEqual(c.val, got),
				"NewRV Src lazy materialize mismatch for %s: want %#v got %#v", c.name, c.val, got)
		})
	}
}

// TestFieldValue_RV_NoBoxOnNewRV asserts NewRV's RV() returns the same reflect.Value
// passed in (rvInit set at build time, never going through reflect.ValueOf(src)).
func TestFieldValue_RV_NoBoxOnNewRV(t *testing.T) {
	x := 42
	rv := reflect.ValueOf(x)
	f := NewRV("", rv)
	// srcSet stays false until Src() is read.
	assert.False(t, f.srcSet)
	got := f.RV()
	assert.Eq(t, reflect.Int, got.Kind())
	assert.Eq(t, int64(42), got.Int())
	// RV() must not have materialized src.
	assert.False(t, f.srcSet)
}

// TestFieldValue_NewRV_InvalidNilRVal verifies the invalid-rv fallback to NilRVal.
func TestFieldValue_NewRV_InvalidNilRVal(t *testing.T) {
	f := NewRV("", reflect.Value{}) // invalid
	rv := f.RV()
	assert.True(t, reflectx.IsNilRV(rv))
	assert.True(t, f.IsEmpty())     // NilObject{} is empty
	s, ok := f.String()             // nil-equivalent -> "", false
	assert.False(t, ok)
	assert.Eq(t, "", s)
	// Src() for invalid -> nil (matches pre-refactor NewRV(invalid) leaving Src=nil).
	assert.Nil(t, f.Src())
}
