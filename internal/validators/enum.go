package validators

import (
	"github.com/gookit/goutil/arrutil"

	"github.com/gookit/validate/v2/internal/fieldval"
	"github.com/gookit/validate/v2/internal/reflectx"
)

// Enum value(int(X),string) should be in the given enum(strings, ints, uints). RV 版。
//
// 逐行对照原 Enum:复用载体缓存 RV() 经 ConvToBasicTypeRV 取基础值;字符串走 []string
// 包含判定,其余按 int64 走 ToInt64s/Int64sHas。原 `if X { return true } return false`
// 与 `return X` 行为等价,此处化简。
func Enum(fl *fieldval.FieldValue, enum any) bool {
	if fl.Src == nil || enum == nil {
		return false
	}

	v, err := reflectx.ConvToBasicTypeRV(fl.RV())
	if err != nil {
		return false
	}

	// if is string value
	if strVal, ok := v.(string); ok {
		if ss, ok := enum.([]string); ok {
			return arrutil.StringsContains(ss, strVal)
		}
		return false
	}

	// as int64 value
	intVal := v.(int64)
	if int64s, err := arrutil.ToInt64s(enum); err == nil {
		return arrutil.Int64sHas(int64s, intVal)
	}
	return false
}

// NotIn value should be not in the given enum(strings, ints, uints). RV 版。
func NotIn(fl *fieldval.FieldValue, enum any) bool { return !Enum(fl, enum) }
