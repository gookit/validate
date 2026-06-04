// Package fieldval holds the internal value carrier (FieldValue) that flows
// through the validating pipeline. It depends only on goutil and the sibling
// internal/reflectx package, never on the validate root package, keeping the
// dependency direction one-way (root -> internal/fieldval).
package fieldval

import (
	"reflect"

	"github.com/gookit/goutil/reflects"

	"github.com/gookit/validate/internal/reflectx"
)

// FieldValue is an internal value carrier that flows through the validating
// pipeline. It holds the raw value together with its lazily-built reflect.Value,
// so the same underlying value is not repeatedly re-reflected (reflect.ValueOf)
// at each pipeline stage (痛点 A / B in docs/validate-v2-design.md §4.3).
//
// All computation is lazy and cached: rv/rt/real and the zero/empty tri-state
// flags are only computed on first access. Designed for single-goroutine use
// within one validation pass (a *Validation is not shared concurrently).
type FieldValue struct {
	name string        // field name/path. eg: "name", "details.sub.*.field"
	Src  any           // original value from the data source
	rv   reflect.Value // = reflect.ValueOf(src), lazy. (nilRVal when src is invalid)
	rt   reflect.Type  // = rv.Type(), lazy
	real reflect.Value // de-pointered rv, lazy

	rvInit   bool // rv/rt initialized
	realInit bool // real initialized
	// tri-state caches: 0=unknown 1=yes 2=no
	zero  int8
	empty int8
}

// New builds a FieldValue from a plain any value (map/form sources).
// The reflect.Value is built lazily on first need.
func New(name string, src any) *FieldValue {
	return &FieldValue{name: name, Src: src}
}

// NewRV builds a FieldValue from an already-available reflect.Value
// (struct source). It avoids an Interface()->ValueOf round-trip. The `src` is
// kept as the boxed value for callers/validators that still consume `any`.
//
// NOTE: reserved for P2 (struct path will pass the cached reflect.Value here);
// defined now so the carrier API is complete.
func NewRV(name string, rv reflect.Value) *FieldValue {
	f := &FieldValue{name: name, rv: rv, rvInit: true}
	if rv.IsValid() {
		f.rt = rv.Type()
		f.Src = rv.Interface()
	} else {
		// keep behavior consistent with rV(): invalid -> nilRVal
		f.rv = reflectx.NilRVal
		f.rt = reflectx.NilRVal.Type()
	}
	return f
}

// RV returns reflect.ValueOf(src), lazily built and cached.
//
// If src is any(nil) the resulting reflect.Value is invalid; in that case it
// returns nilRVal — identical to the fix at callValidatorValue (validating.go),
// so fv.Call() won't panic on an Invalid kind (#125).
func (f *FieldValue) RV() reflect.Value {
	if !f.rvInit {
		rv := reflect.ValueOf(f.Src)
		if !rv.IsValid() {
			rv = reflectx.NilRVal
		}
		f.rv = rv
		f.rt = rv.Type()
		f.rvInit = true
	}
	return f.rv
}

// RT returns the reflect.Type of RV(), lazily built and cached.
func (f *FieldValue) RT() reflect.Type {
	if !f.rvInit {
		f.RV()
	}
	return f.rt
}

// RealV returns the de-pointered reflect.Value.
//
// Semantics are kept byte-for-byte identical to the previous inline handling in
// callValidatorValue (its only P1 consumer): a single non-nil pointer level is
// dereferenced; a nil pointer is left as-is (so typed-nil keeps its pointer
// type). Deeper pointer levels are intentionally NOT unwrapped here to preserve
// 100% behavior equivalence with the pre-refactor code.
func (f *FieldValue) RealV() reflect.Value {
	if !f.realInit {
		rv := f.RV()
		if rv.Kind() == reflect.Ptr && !rv.IsNil() {
			rv = rv.Elem()
		}
		f.real = rv
		f.realInit = true
	}
	return f.real
}

// IsZero reports whether the value is the zero value for its type, matching the
// reflect.Value.IsZero() semantics used by StructData.TryGet. Lazy + cached.
func (f *FieldValue) IsZero() bool {
	if f.zero == 0 {
		rv := f.RV()
		// IsZero panics on an invalid Value; rV() never returns invalid
		// (nilRVal substituted), so this is safe.
		if rv.IsZero() {
			f.zero = 1
		} else {
			f.zero = 2
		}
	}
	return f.zero == 1
}

// IsEmpty reports whether the value is empty, giving results identical to the
// public IsEmpty(any) for the same input — including the untyped-nil and string
// fast paths — but reusing rV() so the value is reflected at most once.
func (f *FieldValue) IsEmpty() bool {
	if f.empty == 0 {
		var emp bool
		switch {
		case f.Src == nil: // untyped nil. same as IsEmpty(nil)
			emp = true
		case isStrSrc(f.Src): // string fast path. same as IsEmpty(string)
			emp = f.Src.(string) == ""
		default:
			emp = reflects.IsEmpty(f.RV())
		}
		if emp {
			f.empty = 1
		} else {
			f.empty = 2
		}
	}
	return f.empty == 1
}

// isStrSrc reports whether src holds a plain string (mirrors the type assertion
// in IsEmpty(any)).
func isStrSrc(src any) bool {
	_, ok := src.(string)
	return ok
}
