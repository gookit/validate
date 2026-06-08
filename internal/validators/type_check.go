package validators

import (
	"reflect"
	"strconv"

	"github.com/gookit/goutil/strutil"

	"github.com/gookit/validate/v2/internal/fieldval"
	"github.com/gookit/validate/v2/internal/reflectx"
)

// 等价契约(本文件 4 个函数共同遵守):
//   新实现(fl) ≡ 旧 public 函数( fl.RealV().Interface() )
// 这是因为这些校验器原走 default → callValidatorValue → reflect.Call,reflect.Call
// 在调 public 前已对值做一次 RealV 预解引用(单层非空指针),public 函数随后再做各自的
// indirection。下面每个实现都复现「RealV 预解引用 + 函数自身 indirection」组合。

// IsUint check, allow: intX, uintX, string. RV 版。
//
// public 无 indirection,对 fl.RealV().Interface() 直接做 type-switch。RealV 已完成
// reflect.Call 路径的那一次预解引用,这里不再二次解引用 —— 与 public(RealV().Interface())
// 逐分支一致。注意用动态类型 type-switch(命名类型 type MyInt int 会落到 int case)。
func IsUint(fl *fieldval.FieldValue) bool {
	switch typVal := fl.RealV().Interface().(type) {
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

// IsBool check. allow: bool, string. RV 版。
//
// public = IndirectValue(val) 后 val.(bool) / val.(string)→strutil.ToBool。复现:对
// fl.RealV()(已预解一层)再做一次 IndirectValue(=第二层),得到的动态类型断言 .(bool)
// /.(string)。⚠️ 用动态类型断言而非 Kind:命名类型 type MyBool bool 会判 false,与 public
// 的 val.(bool) 一致。
func IsBool(fl *fieldval.FieldValue) bool {
	rv := fl.RealV()
	// 第二层 indirection,等价于 public 收到 RealV().Interface() 后自做的 IndirectValue。
	v := reflectx.IndirectValueRV(rv, rv.Interface())

	if _, ok := v.(bool); ok {
		return true
	}
	if typVal, ok := v.(string); ok {
		_, err := strutil.ToBool(typVal)
		return err == nil
	}
	return false
}

// IsArray check value is array or slice. RV 版,支持 strict 变参。
//
// public = if val==nil return false; reflect.Indirect(reflect.ValueOf(val)) 后判 Kind。
// 复现:rv := reflect.Indirect(fl.RealV()) —— fl.RealV() 已解一层,再 Indirect 一层 ＝匹配
// public 的 ValueOf+Indirect 链(含 **T 双指针)。public 的 val==nil 早退对应 fl.Src==nil,
// 但注意 reflect.Call 路径里 public 实际收到 RealV().Interface()(nil 字段为 NilObject{} 非 nil),
// 故等价契约下不在 Src==nil 处早退,而是让 NilObject{}(struct) 自然走到 Kind 判定返回 false。
func IsArray(fl *fieldval.FieldValue, strict ...bool) bool {
	rv := reflect.Indirect(fl.RealV())

	// strict: must be array type.
	if len(strict) > 0 && strict[0] {
		return rv.Kind() == reflect.Array
	}

	// allow array, slice
	return rv.Kind() == reflect.Array || rv.Kind() == reflect.Slice
}

// IsMap check. RV 版。
//
// public = if val==nil return false; reflect.Indirect(reflect.ValueOf(val)).Kind()==Map。
// 复现同 IsArray:rv := reflect.Indirect(fl.RealV()) 后判 Kind==Map。
func IsMap(fl *fieldval.FieldValue) bool {
	return reflect.Indirect(fl.RealV()).Kind() == reflect.Map
}
