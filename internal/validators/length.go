package validators

import (
	"github.com/gookit/goutil/reflects"

	"github.com/gookit/validate/v2/internal/fieldval"
)

// calcLen = CalcLength 的 RV 复用版:复用载体缓存 RV() 免二次 reflect.ValueOf。
// 原 = if val == nil return -1; return reflects.Len(reflect.ValueOf(val))。
func calcLen(fl *fieldval.FieldValue) int {
	if fl.Src == nil {
		return -1
	}
	return reflects.Len(fl.RV())
}

// Length equal check for string, array, slice, map. RV 版。
func Length(fl *fieldval.FieldValue, wantLen int) bool {
	ln := calcLen(fl)
	return ln != -1 && ln == wantLen
}

// MinLength check for string, array, slice, map. RV 版。
func MinLength(fl *fieldval.FieldValue, minLen int) bool {
	ln := calcLen(fl)
	return ln != -1 && ln >= minLen
}

// MaxLength check for string, array, slice, map. RV 版。
func MaxLength(fl *fieldval.FieldValue, maxLen int) bool {
	ln := calcLen(fl)
	return ln != -1 && ln <= maxLen
}
