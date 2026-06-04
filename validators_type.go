package validate

import (
	"reflect"
	"strconv"

	"github.com/gookit/goutil/strutil"
	"github.com/gookit/validate/internal/reflectx"
)

/*************************************************************
 * region global: type validators
 *************************************************************/

// IsUint check, allow: intX, uintX, string
func IsUint(val any) bool {
	switch typVal := val.(type) {
	case int:
		return typVal >= 0
	case int8:
		return typVal >= 0
	case int16:
		return typVal >= 0
	case int32:
		return typVal >= 0
	case int64:
		return typVal >= 0
	case uint, uint8, uint16, uint32, uint64:
		return true
	case string:
		_, err := strconv.ParseUint(typVal, 10, 32)
		return err == nil
	}
	return false
}

// IsBool check. allow: bool, string.
func IsBool(val any) bool {
	val = reflectx.IndirectValue(val)

	if _, ok := val.(bool); ok {
		return true
	}

	if typVal, ok := val.(string); ok {
		_, err := strutil.ToBool(typVal)
		return err == nil
	}
	return false
}

// IsFloat check. allow: floatX, string
func IsFloat(val any) bool {
	val = reflectx.IndirectValue(val)

	if val == nil {
		return false
	}

	switch rv := val.(type) {
	case float32, float64:
		return true
	case string:
		return rv != "" && rxFloat.MatchString(rv)
	}
	return false
}

// IsArray check value is array or slice.
func IsArray(val any, strict ...bool) (ok bool) {
	if val == nil {
		return false
	}

	rv := reflect.Indirect(reflect.ValueOf(val))

	// strict: must go array type.
	if len(strict) > 0 && strict[0] {
		return rv.Kind() == reflect.Array
	}

	// allow array, slice
	return rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice
}

// IsSlice check value is slice type
func IsSlice(val any) (ok bool) {
	if val == nil {
		return false
	}

	rv := reflect.Indirect(reflect.ValueOf(val))
	return rv.Kind() == reflect.Slice
}

// IsInts is int slice check
func IsInts(val any) bool {
	if val == nil {
		return false
	}

	switch val.(type) {
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64:
		return true
	}
	return false
}

// IsStrings is string slice check
func IsStrings(val any) (ok bool) {
	if val == nil {
		return false
	}

	_, ok = val.([]string)
	return
}

// IsMap check
func IsMap(val any) (ok bool) {
	if val == nil {
		return false
	}

	rv := reflect.Indirect(reflect.ValueOf(val))
	return rv.Kind() == reflect.Map
}

// IsInt check, and support length check
func IsInt(val any, minAndMax ...int64) (ok bool) {
	if val == nil {
		return false
	}
	val = reflectx.IndirectValue(val)

	// TODO use mathutil.StrictInt
	intVal, err := valueToInt64(val, true)
	if err != nil {
		return false
	}

	argLn := len(minAndMax)
	if argLn == 0 { // only check type
		return true
	}

	// value check
	minVal := minAndMax[0]
	if argLn == 1 { // only min length check.
		return intVal >= minVal
	}

	// min and max length check
	return intVal >= minVal && intVal <= minAndMax[1]
}

// IsString check and support length check.
//
// Usage:
//
//	ok := IsString(val)
//	ok := IsString(val, 5) // with min len check
//	ok := IsString(val, 5, 12) // with min and max len check
func IsString(val any, minAndMaxLen ...int) (ok bool) {
	val = reflectx.IndirectValue(val)

	if val == nil {
		return false
	}

	argLn := len(minAndMaxLen)
	str, isStr := val.(string)

	// only check type
	if argLn == 0 {
		return isStr
	}

	if !isStr {
		return false
	}

	// length check
	strLen := len(str)
	minLen := minAndMaxLen[0]

	// only min length check.
	if argLn == 1 {
		return strLen >= minLen
	}

	// min and max length check
	return strLen >= minLen && strLen <= minAndMaxLen[1]
}
