package validate

import (
	"reflect"
	"sync"
	"sync/atomic"
)

// CustomTypeFunc 从自定义类型字段提取 validate 实际要校验的底层值。
//
// 返回 nil 表示"视为空/未设置"(required 失败)。提取后的值天然走现有
// valueCompare / IsEmpty / Length / string 校验路径。对标
// go-playground/validator 的 RegisterCustomTypeFunc。
type CustomTypeFunc func(field reflect.Value) any

// customTypes 自定义类型提取器注册表。
//
// key=reflect.Type, value=CustomTypeFunc。读多写少:校验热路径只读,
// 注册仅在初始化时发生,故使用 sync.Map(无锁读)最合适。
var customTypes sync.Map // map[reflect.Type]CustomTypeFunc

// hasCustomTypes 零开销门控:未注册任何自定义类型时为 false,校验热路径
// 仅需一次 atomic.Bool load 即可短路,避免 sync.Map 查找开销。
var hasCustomTypes atomic.Bool

// AddCustomType 为给定样例类型注册底层值提取器(命名与 AddValidator 一致)。
//
// 按传入样例的精确 reflect.Type 存储,不自动解指针:传 sql.NullString{} 只
// 匹配该值类型;若要同时匹配指针,需另外传入 &sql.NullString{} 样例。
func AddCustomType(fn CustomTypeFunc, types ...any) {
	if fn == nil || len(types) == 0 {
		return
	}

	for _, sample := range types {
		if sample == nil {
			continue
		}
		customTypes.Store(reflect.TypeOf(sample), fn)
	}
	hasCustomTypes.Store(true)
}

// ResetCustomTypes 清空所有已注册的自定义类型提取器并复位门控(测试/清理用,
// 参照 ResetTypeCache)。通过 Range+Delete 实现以保持并发安全。
func ResetCustomTypes() {
	customTypes.Range(func(key, _ any) bool {
		customTypes.Delete(key)
		return true
	})
	hasCustomTypes.Store(false)
}

// resolveCustomType 尝试将 val 提取为底层可校验值。
//
//   - 门控为 false 时直接返回 (val, false),保证未注册时零额外开销。
//   - val 为 nil 时安全返回 (val, false)。
//   - 命中注册类型则调用提取器,返回 (extracted, true);未命中返回 (val, false)。
func resolveCustomType(val any) (any, bool) {
	if !hasCustomTypes.Load() || val == nil {
		return val, false
	}

	if fn, ok := customTypes.Load(reflect.TypeOf(val)); ok {
		return fn.(CustomTypeFunc)(reflect.ValueOf(val)), true
	}
	return val, false
}
