package validators

import (
	"github.com/gookit/goutil/reflects"

	"github.com/gookit/validate/v2/internal/fieldval"
	"github.com/gookit/validate/v2/internal/reflectx"
)

// calcLen = CalcLength 的 RV 复用版:复用载体缓存 RV() 免二次 reflect.ValueOf。
// 纯 RV、不读 Src:nil 字段经 RV() 物化为 NilRVal(NilObject{}),显式判 NilRVal 返回 -1,
// 与现 `Src==nil → -1` 字节级等价(reflects.Len(NilObject{}) 走 default 同样返回 -1)。
func calcLen(fl *fieldval.FieldValue) int {
	rv := fl.RV()
	if reflectx.IsNilRV(rv) {
		return -1
	}
	return reflects.Len(rv)
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
