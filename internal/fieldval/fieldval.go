// Package fieldval holds the internal value carrier (FieldValue) that flows
// through the validating pipeline. It depends only on goutil and the sibling
// internal/reflectx package, never on the validate root package, keeping the
// dependency direction one-way (root -> internal/fieldval).
package fieldval

import (
	"reflect"

	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"

	"github.com/gookit/validate/v2/internal/reflectx"
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
	src  any           // original value from the data source (lazily materialized for NewRV)
	rv   reflect.Value // = reflect.ValueOf(src), lazy. (nilRVal when src is invalid)
	rt   reflect.Type  // = rv.Type(), lazy
	real reflect.Value // de-pointered rv, lazy

	srcSet   bool // src materialized (New: eager; NewRV: lazy until Src() called)
	rvInit   bool // rv/rt initialized
	realInit bool // real initialized
	// tri-state caches: 0=unknown 1=yes 2=no
	zero  int8
	empty int8
}

// New builds a FieldValue from a plain any value (map/form sources).
// The caller already holds the boxed value, so src is set eagerly; the
// reflect.Value is still built lazily on first need.
func New(name string, src any) *FieldValue {
	return &FieldValue{name: name, src: src, srcSet: true}
}

// NewRV builds a FieldValue from an already-available reflect.Value
// (struct source). It avoids an Interface()->ValueOf round-trip AND keeps the
// boxed value lazy: src is materialized (rv.Interface()) only on first Src()
// call, so RV-native consumers (IsEmpty/String/calcLen/required) never box.
func NewRV(name string, rv reflect.Value) *FieldValue {
	f := &FieldValue{name: name, rv: rv, rvInit: true}
	if rv.IsValid() {
		f.rt = rv.Type()
		// srcSet stays false: defer rv.Interface() boxing until Src() is read.
	} else {
		// keep behavior consistent with RV(): invalid -> nilRVal. src is pinned
		// to nil here (srcSet=true) so Src() returns nil, byte-for-byte matching
		// the pre-refactor NewRV(invalid) which left Src as its zero value (nil).
		f.rv = reflectx.NilRVal
		f.rt = reflectx.NilRVal.Type()
		f.src = nil
		f.srcSet = true
	}
	return f
}

// Src returns the original boxed value, materializing it lazily for NewRV
// carriers (only reached when a downstream consumer truly needs `any`). For New
// carriers src was set eagerly; for NewRV(invalid) src was pinned to nil at
// construction. So the remaining lazy case here is NewRV(valid): rv.Interface(),
// matching the previous eager `f.Src = rv.Interface()` exactly.
func (f *FieldValue) Src() any {
	if !f.srcSet {
		f.src = f.rv.Interface()
		f.srcSet = true
	}
	return f.src
}

// RV returns reflect.ValueOf(src), lazily built and cached.
//
// If src is any(nil) the resulting reflect.Value is invalid; in that case it
// returns nilRVal — identical to the fix at callValidatorValue (validating.go),
// so fv.Call() won't panic on an Invalid kind (#125).
func (f *FieldValue) RV() reflect.Value {
	if !f.rvInit {
		// only reached by New carriers (NewRV sets rvInit=true at build time),
		// where srcSet is true, so f.src is read directly — no Src() boxing.
		rv := reflect.ValueOf(f.src)
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
// public IsEmpty(any) for the same input — without ever reading the boxed src.
//
// Pure reflect.Value: RV() substitutes NilRVal (= reflect.ValueOf(NilObject{}))
// for an untyped-nil src, and reflects.IsEmpty(NilRVal) returns true via its
// DeepEqual default branch — matching public IsEmpty(nil)==true. A string src
// surfaces as Kind String, where reflects.IsEmpty does v.Len()==0, matching the
// `s == ""` fast path. So one call covers nil / string / everything.
func (f *FieldValue) IsEmpty() bool {
	if f.empty == 0 {
		if reflects.IsEmpty(f.RV()) {
			f.empty = 1
		} else {
			f.empty = 2
		}
	}
	return f.empty == 1
}

// String 返回字段值的字符串形式,语义与 validating.go valToString 字节级一致,
// 但复用缓存的 RV()(命名字符串类型/可转换值都不 panic)。bool 报告是否取到可用字符串。
//
// NOTE: 签名是 () (string, bool),不满足 fmt.Stringer,无冲突。
func (f *FieldValue) String() (string, bool) {
	rv := f.RV()
	// 命名字符串类型也是 Kind String,直接取值,box-free。涵盖 plain string 快路径。
	if rv.Kind() == reflect.String {
		return rv.String(), true
	}
	// untyped nil src 经 RV() 物化为 NilRVal(NilObject{}),与现 `f.Src==nil`
	// 分支等价:返回 "", false(无可用字符串)。box-free 的 Type 比较检测。
	if reflectx.IsNilRV(rv) {
		return "", false
	}
	// 非字符串值才物化(经 Src() 懒装箱),交给 strutil.ToString,与现 valToString 一致。
	s, err := strutil.ToString(f.Src())
	return s, err == nil
}

// Indirect 返回单层指针解引用后的值(any),语义与 reflectx.IndirectValue(Src) 字节级一致:
// 未类型化 nil→nil;非指针→原样(Src,无新装箱);非空指针→解引用后的 Interface();空指针→nil。
// 复用缓存 RV(),避免二次 reflect.ValueOf。
func (f *FieldValue) Indirect() any {
	// Indirect 返回 any 必然装箱(非 bench 热路径,本步不优化),仅把字段读改为
	// Src() 方法以适配懒化;nil 检测与逻辑保持与改造前字节级一致。
	if f.Src() == nil {
		return nil
	}
	rv := f.RV()
	if rv.Kind() == reflect.Ptr {
		if ev := reflect.Indirect(rv); ev.IsValid() {
			return ev.Interface()
		}
		return nil
	}
	return f.Src()
}
