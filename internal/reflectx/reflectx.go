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
	v := reflect.Indirect(reflect.ValueOf(val))

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
