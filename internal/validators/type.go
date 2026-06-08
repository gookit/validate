package validators

import (
	"reflect"

	"github.com/gookit/goutil/mathutil"

	"github.com/gookit/validate/v2/internal/fieldval"
)

// IsSlice check value is slice type. RV 版,复用载体缓存 RV() 免二次 reflect.ValueOf。
//
// 原 = if val == nil return false; reflect.Indirect(reflect.ValueOf(val)).Kind() == Slice
func IsSlice(fl *fieldval.FieldValue) bool {
	if fl.Src == nil {
		return false
	}
	return reflect.Indirect(fl.RV()).Kind() == reflect.Slice
}

// IsInt check, and support length check. RV 版。
//
// 原 = if val == nil return false; val = IndirectValue(val); mathutil.StrictInt(val);
// 然后按 minAndMax 个数做长度判定。nil 判定在 Indirect 之前(与原函数顺序一致)。
func IsInt(fl *fieldval.FieldValue, minAndMax ...int64) bool {
	if fl.Src == nil {
		return false
	}

	intVal, valid := mathutil.StrictInt(fl.Indirect())
	if !valid {
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

// IsString check and support length check. RV 版。
//
// 原 = val = IndirectValue(val); if val == nil return false; str,isStr := val.(string);
// 然后按 minAndMaxLen 个数做长度判定。nil 判定在 Indirect 之后(与原函数顺序一致)。
func IsString(fl *fieldval.FieldValue, minAndMaxLen ...int) bool {
	v := fl.Indirect()
	if v == nil {
		return false
	}

	argLn := len(minAndMaxLen)
	str, isStr := v.(string)

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
