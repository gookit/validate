package validators

import (
	"reflect"

	"github.com/gookit/goutil/arrutil"

	"github.com/gookit/validate/v2/internal/fieldval"
	"github.com/gookit/validate/v2/internal/reflectx"
)

// Enum value(int(X),string) should be in the given enum(strings, ints, uints). RV 版。
//
// 逐行对照原 Enum:复用载体缓存 RV() 经 ConvToBasicTypeRV 取基础值;字符串走 []string
// 包含判定,其余按 int64 走 ToInt64s/Int64sHas。
//
// A slice or array value (eg. a multi-select field) is in the enum when every
// element is in it, so `in:go,php` accepts ["go","php"] and rejects ["go","java"].
func Enum(fl *fieldval.FieldValue, enum any) bool {
	if fl.Src() == nil || enum == nil {
		return false
	}

	return eachBasicValue(fl, func(v any) bool { return enumHas(enum, v) })
}

// NotIn value should be not in the given enum(strings, ints, uints). RV 版。
//
// For a slice or array value, no element may be in the enum, so `not_in:go,php`
// rejects ["go","java"] but accepts ["java","rust"].
func NotIn(fl *fieldval.FieldValue, enum any) bool {
	if fl.Src() == nil || enum == nil {
		return false
	}

	return eachBasicValue(fl, func(v any) bool { return !enumHas(enum, v) })
}

// EnumSupported reports whether the value can be checked by the enum rules. Only a map is
// reported as unsupported: the enum rules compare a value (or the elements of a slice)
// with the given list, a map value has nothing to compare and used to fail with the
// rule message, which reads like the value is simply not in the list. Any other value
// stays with the old behaviour, including a struct reached through a nil nested pointer,
// which the rules already treat as an unset value.
func EnumSupported(rv reflect.Value) bool {
	rv = derefRV(rv)
	if !rv.IsValid() {
		return true
	}

	return rv.Kind() != reflect.Map
}

// derefRV unwraps pointer/interface values, an invalid Value is returned for a nil one.
func derefRV(rv reflect.Value) reflect.Value {
	for rv.IsValid() && (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return reflect.Value{}
		}
		rv = rv.Elem()
	}
	return rv
}

// eachBasicValue calls fn with the basic value of the field: the value itself for a
// scalar field, or every element for a slice/array field. It stops and returns false
// as soon as fn returns false, or when a value cannot be converted to a basic type.
func eachBasicValue(fl *fieldval.FieldValue, fn func(v any) bool) bool {
	rv := fl.RV()
	if kind := rv.Kind(); kind != reflect.Slice && kind != reflect.Array {
		v, err := reflectx.ConvToBasicTypeRV(rv)
		if err != nil {
			return false
		}
		return fn(v)
	}

	for i := 0; i < rv.Len(); i++ {
		ev := rv.Index(i)
		// unwrap an element pointer, a nil one has no value to check
		for ev.Kind() == reflect.Ptr {
			if ev.IsNil() {
				ev = reflect.Value{}
				break
			}
			ev = ev.Elem()
		}
		if !ev.IsValid() {
			continue
		}

		v, err := reflectx.ConvToBasicTypeRV(ev)
		if err != nil {
			return false
		}
		if !fn(v) {
			return false
		}
	}
	return true
}

// enumHas reports whether the basic value (string or int64) is in the given enum.
func enumHas(enum any, v any) bool {
	// if is string value
	if strVal, ok := v.(string); ok {
		if ss, ok := enum.([]string); ok {
			return arrutil.StringsContains(ss, strVal)
		}
		return false
	}

	// as int64 value
	intVal, ok := v.(int64)
	if !ok {
		return false
	}
	if int64s, err := arrutil.ToInt64s(enum); err == nil {
		return arrutil.Int64sHas(int64s, intVal)
	}
	return false
}
