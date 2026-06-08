package validate

import (
	"bytes"
	"reflect"
	"time"
	"unicode/utf8"

	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/validate/v2/internal/fieldval"
	"github.com/gookit/validate/v2/internal/reflectx"
	ivalidators "github.com/gookit/validate/v2/internal/validators"
)

/*************************************************************
 * region global: filesystem validators
 *************************************************************/

// PathExists reports whether the named file or directory exists.
func PathExists(path string) bool { return fsutil.PathExists(path) }

// IsFilePath path is a local filepath
func IsFilePath(path string) bool { return fsutil.IsFile(path) }

// IsDirPath path is a local dir path
func IsDirPath(path string) bool { return fsutil.IsDir(path) }

// IsWinPath string
func IsWinPath(s string) bool {
	return s != "" && rxWinPath.MatchString(s)
}

// IsUnixPath string
func IsUnixPath(s string) bool {
	return s != "" && rxUnixPath.MatchString(s)
}

/*************************************************************
 * global: compare validators
 *************************************************************/

// IsEqual check two value is equals. Don't compare func, struct
//
// Support:
//
//	bool, int(X), uint(X), string, float(X) AND slice, array, map
func IsEqual(val, wantVal any) bool {
	// check is nil
	if val == nil || wantVal == nil {
		return val == wantVal
	}

	sv := reflectx.RemoveValuePtr(reflect.ValueOf(val))
	wv := reflectx.RemoveValuePtr(reflect.ValueOf(wantVal))

	// don't compare func, struct
	if sv.Kind() == reflect.Func || sv.Kind() == reflect.Struct {
		return false
	}
	if wv.Kind() == reflect.Func || wv.Kind() == reflect.Struct {
		return false
	}

	// compare basic type: bool, int(X), uint(X), string, float(X)
	equal, err := eq(sv, wv)

	// is not a basic type, eg: slice, array, map ...
	if err != nil {
		expBt, ok := val.([]byte)
		if !ok {
			return reflect.DeepEqual(val, wantVal)
		}

		actBt, ok := wantVal.([]byte)
		if !ok {
			return false
		}
		if expBt == nil || actBt == nil {
			return expBt == nil && actBt == nil
		}

		return bytes.Equal(expBt, actBt)
	}

	return equal
}

// NotEqual check
func NotEqual(val, wantVal any) bool { return !IsEqual(val, wantVal) }

// IntEqual check
func IntEqual(val any, wantVal int64) bool {
	// intVal, isInt := IntVal(val)
	intVal, err := mathutil.Int64(val)
	if err != nil {
		return false
	}

	return intVal == wantVal
}

// Gt check value greater dst value.
//
// only check for: int(X), uint(X), float(X), string.
func Gt(val, minVal any) bool { return ivalidators.Gt(fieldval.New("", val), minVal) }

// Gte check value greater or equal dst value
// only check for: int(X), uint(X), float(X), string.
func Gte(val, minVal any) bool { return ivalidators.Gte(fieldval.New("", val), minVal) }

// Min check value greater or equal dst value, alias Gte()
// only check for: int(X), uint(X), float(X), string.
func Min(val, minVal any) bool { return ivalidators.Min(fieldval.New("", val), minVal) }

// Lt less than dst value.
// only check for: int(X), uint(X), float(X).
func Lt(val, maxVal any) bool { return ivalidators.Lt(fieldval.New("", val), maxVal) }

// Lte less than or equal dst value.
// only check for: int(X), uint(X), float(X).
func Lte(val, maxVal any) bool { return ivalidators.Lte(fieldval.New("", val), maxVal) }

// Max less than or equal dst value, alias Lte()
// only check for: int(X), uint(X), float(X).
func Max(val, maxVal any) bool { return ivalidators.Max(fieldval.New("", val), maxVal) }

// Between value in the given range (inclusive).
// only check for: int(X), uint(X), float(X), string.
func Between(val, minVal, maxVal any) bool {
	return ivalidators.Between(fieldval.New("", val), minVal, maxVal)
}

/*************************************************************
 * region global: array, slice, map validators
 *************************************************************/

// Enum value(int(X),string) should be in the given enum(strings, ints, uints).
func Enum(val, enum any) bool {
	if val == nil || enum == nil {
		return false
	}

	v, err := reflectx.ConvToBasicType(val)
	if err != nil {
		return false
	}

	// if is string value
	if strVal, ok := v.(string); ok {
		if ss, ok := enum.([]string); ok {
			if arrutil.StringsContains(ss, strVal) {
				return true
			}
		}
		return false
	}

	// as int64 value
	intVal := v.(int64)
	if int64s, err := arrutil.ToInt64s(enum); err == nil {
		if arrutil.Int64sHas(int64s, intVal) {
			return true
		}
	}
	return false
}

// NotIn value should be not in the given enum(strings, ints, uints).
func NotIn(val, enum any) bool { return !Enum(val, enum) }

/*************************************************************
 * region global: length validators
 *************************************************************/

// Length equal check for string, array, slice, map
func Length(val any, wantLen int) bool { return ivalidators.Length(fieldval.New("", val), wantLen) }

// MinLength check for string, array, slice, map
func MinLength(val any, minLen int) bool {
	return ivalidators.MinLength(fieldval.New("", val), minLen)
}

// MaxLength check for string, array, slice, map
func MaxLength(val any, maxLen int) bool {
	return ivalidators.MaxLength(fieldval.New("", val), maxLen)
}

// ByteLength check string's length
func ByteLength(str string, minLen int, maxLen ...int) bool {
	strLen := len(str)

	// only min length check.
	if len(maxLen) == 0 {
		return strLen >= minLen
	}

	// min and max length check
	return strLen >= minLen && strLen <= maxLen[0]
}

// RuneLength check string's length (including multibyte strings)
func RuneLength(val any, minLen int, maxLen ...int) bool {
	str, isString := val.(string)
	if !isString {
		return false
	}

	// strLen := len([]rune(str))
	strLen := utf8.RuneCountInString(str)

	// only min length check.
	if len(maxLen) == 0 {
		return strLen >= minLen
	}

	// min and max length check
	return strLen >= minLen && strLen <= maxLen[0]
}

// StringLength check string's length (including multibyte strings)
func StringLength(val any, minLen int, maxLen ...int) bool {
	return RuneLength(val, minLen, maxLen...)
}

/*************************************************************
 * global: date/time validators
 *************************************************************/

// IsDate check value is an date string.
func IsDate(srcDate string, layouts ...string) bool {
	_, err := strutil.ToTime(srcDate, layouts...)
	return err == nil
}

// DateFormat check
func DateFormat(s string, layout string) bool {
	_, err := time.Parse(layout, s)
	return err == nil
}

// DateEquals check.
// Usage:
// 	DateEquals(val, "2017-05-12")
// func DateEquals(srcDate, dstDate string) bool {
// 	return false
// }

// BeforeDate check
func BeforeDate(srcDate, dstDate string) bool {
	st, err := strutil.ToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := strutil.ToTime(dstDate)
	if err != nil {
		return false
	}

	return st.Before(dt)
}

// BeforeOrEqualDate check
func BeforeOrEqualDate(srcDate, dstDate string) bool {
	st, err := strutil.ToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := strutil.ToTime(dstDate)
	if err != nil {
		return false
	}

	return st.Before(dt) || st.Equal(dt)
}

// AfterOrEqualDate check
func AfterOrEqualDate(srcDate, dstDate string) bool {
	st, err := strutil.ToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := strutil.ToTime(dstDate)
	if err != nil {
		return false
	}

	return st.After(dt) || st.Equal(dt)
}

// AfterDate check
func AfterDate(srcDate, dstDate string) bool {
	st, err := strutil.ToTime(srcDate)
	if err != nil {
		return false
	}

	dt, err := strutil.ToTime(dstDate)
	if err != nil {
		return false
	}

	return st.After(dt)
}
