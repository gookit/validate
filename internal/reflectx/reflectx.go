// Package reflectx holds pure reflection helpers extracted from the validate
// root package. It depends only on goutil and never imports the root package,
// keeping the dependency direction one-way (root -> internal/reflectx).
package reflectx

import (
	"errors"
	"reflect"

	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
)

// ErrConvertFail error. Mirrors validate.ErrConvertFail; kept as a separate
// value because consumers only test err != nil, never the identity.
var ErrConvertFail = errors.New("convert value is failure")

// NilObject represent nil value for calling functions and should be reflected at custom filters as nil variable.
//
// NOTE: validate.NilObject is a type alias of this type, so the public API and
// any val.(NilObject) assertion in user code keep working unchanged.
type NilObject struct{}

// nilObj zero value of NilObject.
var nilObj = NilObject{}

// NilRVal a reflect nil value (= reflect.ValueOf(NilObject{})).
var NilRVal = reflect.ValueOf(nilObj)

// nilRType is the reflect.Type of NilObject, cached for box-free IsNilRV checks.
var nilRType = NilRVal.Type()

// IsNilObj check value is internal NilObject
func IsNilObj(val any) bool {
	_, ok := val.(NilObject)
	return ok
}

// IsNilRV reports whether rv is the nil sentinel — an invalid Value or a
// NilObject — using a box-free Type comparison (no rv.Interface()). The carrier
// substitutes NilRVal for an untyped-nil src, so this detects "nil field" purely
// from the reflect.Value.
func IsNilRV(rv reflect.Value) bool {
	return !rv.IsValid() || rv.Type() == nilRType
}

// ValueCompare value compare.
//
// only check for: int(X), uint(X), float(X), string.
func ValueCompare(srcVal, dstVal any, op string) (ok bool) {
	srcVal = IndirectValue(srcVal)

	// string compare
	if str1, ok := srcVal.(string); ok {
		str2, err := strutil.ToString(dstVal)
		if err != nil {
			return false
		}

		return strutil.Compare(str1, str2, op)
	}

	// as int or float to compare
	return mathutil.Compare(srcVal, dstVal, op)
}

// GetVariadicKind name.
//
// usage:
//
//	GetVariadicKind(reflect.TypeOf(v))
func GetVariadicKind(typ reflect.Type) reflect.Kind {
	if typ.Kind() == reflect.Slice {
		return typ.Elem().Kind()
	}
	return reflect.Invalid
}

// ConvTypeByBaseKind convert value type by base kind
//
//nolint:forcetypeassert
func ConvTypeByBaseKind(srcVal any, dstType reflect.Kind) (any, error) {
	rv, err := reflects.ConvToKind(srcVal, dstType)
	if err != nil {
		return nil, err
	}
	return rv.Interface(), nil
}

// ConvToBasicType convert custom type to generic basic int, string, unit.
// returns string, int64 or error
func ConvToBasicType(val any) (value any, err error) {
	return ConvToBasicTypeRV(reflect.ValueOf(val))
}

// ConvToBasicTypeRV 同 ConvToBasicType,但直接吃 reflect.Value(复用调用方缓存,
// 免二次 reflect.ValueOf)。语义与 ConvToBasicType 字节级一致。
func ConvToBasicTypeRV(rv reflect.Value) (value any, err error) {
	v := reflect.Indirect(rv)

	switch v.Kind() {
	case reflect.String:
		value = v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = int64(v.Uint()) // always return int64
	default:
		err = ErrConvertFail
	}
	return
}

// RemoveValuePtr removes value multiple pointer
func RemoveValuePtr(t reflect.Value) reflect.Value {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// IndirectValue dereferences a single pointer level of the input value.
func IndirectValue(input any) any {
	// Check if input is nil
	if input == nil {
		return nil
	}

	// Use reflect to handle the value
	val := reflect.ValueOf(input)

	// If the value is a pointer, then use reflect.Indirect to get the actual value it points to
	if val.Kind() == reflect.Ptr {
		// Use reflect.Indirect to safely dereference the pointer
		val = reflect.Indirect(val)

		// If the dereferenced value is valid (not nil), return the interface
		if val.IsValid() {
			return val.Interface()
		}
		// If the dereferenced value is not valid, return nil
		return nil
	}

	// If the input is not a pointer, just return the input as it is
	return input
}

// IndirectValueRV is the reflect.Value flavour of IndirectValue: it dereferences
// a single pointer level of rv and returns the value as any. It mirrors
// IndirectValue's pointer / nil-pointer / non-pointer branches; orig is the boxed
// value rv was reflected from and is returned as-is for the non-pointer branch
// (no fresh boxing alloc — and identical to IndirectValue, which returns the
// original input unchanged for non-pointers).
//
// NOTE: callers pass a carrier's RealV() here (already de-pointered one level),
// so this applies the SECOND indirection level — matching the reflect.Call path
// where the public IsBool receives RealV().Interface() and then does its own
// IndirectValue. RealV() substitutes NilRVal for a nil src (never an invalid
// Value), so a nil field surfaces as NilObject{} not nil — same as reflect.Call.
func IndirectValueRV(rv reflect.Value, orig any) any {
	// If the value is a pointer, then use reflect.Indirect to get the actual value it points to
	if rv.Kind() == reflect.Ptr {
		// Use reflect.Indirect to safely dereference the pointer
		rv = reflect.Indirect(rv)

		// If the dereferenced value is valid (not nil), return the interface
		if rv.IsValid() {
			return rv.Interface()
		}
		// If the dereferenced value is not valid, return nil
		return nil
	}

	// If the input is not a pointer, just return the original value as it is.
	return orig
}
