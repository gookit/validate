package validate

import "reflect"

// fieldValue is an internal value carrier that flows through the validating
// pipeline. It holds the raw value together with its lazily-built reflect.Value,
// so the same underlying value is not repeatedly re-reflected (reflect.ValueOf)
// at each pipeline stage (痛点 A / B in docs/validate-v2-design.md §4.3).
//
// All computation is lazy and cached: rv/rt/real and the zero/empty tri-state
// flags are only computed on first access. Designed for single-goroutine use
// within one validation pass (a *Validation is not shared concurrently).
type fieldValue struct {
	name string        // field name/path. eg: "name", "details.sub.*.field"
	src  any           // original value from the data source
	rv   reflect.Value // = reflect.ValueOf(src), lazy. (nilRVal when src is invalid)
	rt   reflect.Type  // = rv.Type(), lazy
	real reflect.Value // de-pointered rv, lazy

	rvInit   bool // rv/rt initialized
	realInit bool // real initialized
	// tri-state caches: 0=unknown 1=yes 2=no
	zero  int8
	empty int8
}

// newFieldValue builds a fieldValue from a plain any value (map/form sources).
// The reflect.Value is built lazily on first need.
func newFieldValue(name string, src any) *fieldValue {
	return &fieldValue{name: name, src: src}
}

// newFieldValueRV builds a fieldValue from an already-available reflect.Value
// (struct source). It avoids an Interface()->ValueOf round-trip. The `src` is
// kept as the boxed value for callers/validators that still consume `any`.
//
// NOTE: reserved for P2 (struct path will pass the cached reflect.Value here);
// defined now so the carrier API is complete.
func newFieldValueRV(name string, rv reflect.Value) *fieldValue {
	f := &fieldValue{name: name, rv: rv, rvInit: true}
	if rv.IsValid() {
		f.rt = rv.Type()
		f.src = rv.Interface()
	} else {
		// keep behavior consistent with rV(): invalid -> nilRVal
		f.rv = nilRVal
		f.rt = nilRVal.Type()
	}
	return f
}

// rV returns reflect.ValueOf(src), lazily built and cached.
//
// If src is any(nil) the resulting reflect.Value is invalid; in that case it
// returns nilRVal — identical to the fix at callValidatorValue (validating.go),
// so fv.Call() won't panic on an Invalid kind (#125).
func (f *fieldValue) rV() reflect.Value {
	if !f.rvInit {
		rv := reflect.ValueOf(f.src)
		if !rv.IsValid() {
			rv = nilRVal
		}
		f.rv = rv
		f.rt = rv.Type()
		f.rvInit = true
	}
	return f.rv
}

// rT returns the reflect.Type of rV(), lazily built and cached.
func (f *fieldValue) rT() reflect.Type {
	if !f.rvInit {
		f.rV()
	}
	return f.rt
}

// realV returns the de-pointered reflect.Value.
//
// Semantics are kept byte-for-byte identical to the previous inline handling in
// callValidatorValue (its only P1 consumer): a single non-nil pointer level is
// dereferenced; a nil pointer is left as-is (so typed-nil keeps its pointer
// type). Deeper pointer levels are intentionally NOT unwrapped here to preserve
// 100% behavior equivalence with the pre-refactor code.
func (f *fieldValue) realV() reflect.Value {
	if !f.realInit {
		rv := f.rV()
		if rv.Kind() == reflect.Ptr && !rv.IsNil() {
			rv = rv.Elem()
		}
		f.real = rv
		f.realInit = true
	}
	return f.real
}

// isZero reports whether the value is the zero value for its type, matching the
// reflect.Value.IsZero() semantics used by StructData.TryGet. Lazy + cached.
func (f *fieldValue) isZero() bool {
	if f.zero == 0 {
		rv := f.rV()
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

// isEmpty reports whether the value is empty, giving results identical to the
// public IsEmpty(any) for the same input — including the untyped-nil and string
// fast paths — but reusing rV() so the value is reflected at most once.
func (f *fieldValue) isEmpty() bool {
	if f.empty == 0 {
		var emp bool
		switch {
		case f.src == nil: // untyped nil. same as IsEmpty(nil)
			emp = true
		case isStrSrc(f.src): // string fast path. same as IsEmpty(string)
			emp = f.src.(string) == ""
		default:
			emp = ValueIsEmpty(f.rV())
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
