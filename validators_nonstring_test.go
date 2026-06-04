package validate

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// namedStr 是一个底层为 string 的具名类型：其 reflect.Kind == String，
// 因此在 valueValidate 中会跳过 val->string 的预转换，旧实现 val.(string)
// 会直接 panic。这里用它复现并验证修复。
type namedStr string

// TestStringValidators_NonStringValue 覆盖 callValidator switch 中
// regexp/isJSON/isStringNumber 三个原本对 val 直接做 val.(string) 断言的分支：
// 对非 string 字段(int、具名 string、不可字符串化类型)施加这些规则时，
// 必须不 panic，且结果合理(可字符串化 -> 正常校验；不可字符串化 -> 校验失败)。
func TestStringValidators_NonStringValue(t *testing.T) {
	// validate 一个 map 字段 + 一条 string 规则，返回是否通过(不抛 panic)。
	run := func(val any, rule string) bool {
		v := Map(map[string]any{"f": val})
		v.StringRule("f", rule)
		return v.Validate()
	}

	t.Run("regexp on int value", func(t *testing.T) {
		// 旧实现: val.(int) -> panic. 新实现: ToString(123)="123", 匹配 \d+ -> true
		var pass bool
		assert.NotPanics(t, func() { pass = run(123, `regexp:\d+`) })
		assert.True(t, pass)

		assert.NotPanics(t, func() { pass = run(123, `regexp:^[a-z]+$`) })
		assert.False(t, pass)
	})

	t.Run("regexp on named string value", func(t *testing.T) {
		// 具名 string: Kind==String 会跳过预转换, 旧实现 val.(string) panic
		var pass bool
		assert.NotPanics(t, func() { pass = run(namedStr("12345"), `regexp:\d+`) })
		assert.True(t, pass)
	})

	t.Run("isJSON on int value", func(t *testing.T) {
		// int 转 "123" 不是合法 JSON 对象/数组 -> false, 但不 panic
		var pass bool
		assert.NotPanics(t, func() { pass = run(123, "isJSON") })
		assert.False(t, pass)
	})

	t.Run("isJSON on named string value", func(t *testing.T) {
		var pass bool
		assert.NotPanics(t, func() { pass = run(namedStr(`{"a":1}`), "isJSON") })
		assert.True(t, pass)
	})

	t.Run("isJSON on non-stringifiable value", func(t *testing.T) {
		// map/slice 无法被 strutil.ToString 转为 string -> valToString 返回 false
		// -> 校验失败(false), 但不 panic
		var pass bool
		assert.NotPanics(t, func() { pass = run(map[string]int{"a": 1}, "isJSON") })
		assert.False(t, pass)
	})

	t.Run("isStringNumber on int value", func(t *testing.T) {
		// "123" 是合法的数字字符串 -> true
		var pass bool
		assert.NotPanics(t, func() { pass = run(123, "isStringNumber") })
		assert.True(t, pass)
	})

	t.Run("isStringNumber on named string value", func(t *testing.T) {
		var pass bool
		assert.NotPanics(t, func() { pass = run(namedStr("123"), "isStringNumber") })
		assert.True(t, pass)

		assert.NotPanics(t, func() { pass = run(namedStr("abc"), "isStringNumber") })
		assert.False(t, pass)
	})

	t.Run("isStringNumber on bool value", func(t *testing.T) {
		// bool -> "true"/"false", 不是数字字符串 -> false, 不 panic
		var pass bool
		assert.NotPanics(t, func() { pass = run(true, "isStringNumber") })
		assert.False(t, pass)
	})
}

// TestValToString 直接覆盖 valToString helper 的关键分支。
func TestValToString(t *testing.T) {
	t.Run("plain string returned byte-for-byte", func(t *testing.T) {
		s, ok := valToString("hello")
		assert.True(t, ok)
		assert.Eq(t, "hello", s)
	})

	t.Run("named string via reflect", func(t *testing.T) {
		s, ok := valToString(namedStr("xyz"))
		assert.True(t, ok)
		assert.Eq(t, "xyz", s)
	})

	t.Run("int coerced", func(t *testing.T) {
		s, ok := valToString(123)
		assert.True(t, ok)
		assert.Eq(t, "123", s)
	})

	t.Run("nil not stringifiable", func(t *testing.T) {
		_, ok := valToString(nil)
		assert.False(t, ok)
	})

	t.Run("map not stringifiable", func(t *testing.T) {
		_, ok := valToString(map[string]int{"a": 1})
		assert.False(t, ok)
	})
}
