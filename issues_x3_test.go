package validate_test

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2"
)

// https://github.com/gookit/validate/v2/issues/184
// Improve error messages: add original value that caused an error
//
// 诉求: 同时校验大量规则时, 错误消息看不出是哪个值触发的, 希望能在消息里带上原始值。
//
// 实现: opt-in 开关 GlobalOption.ErrShowValue (默认 false), 会拷贝到每个 Validation
// 实例。默认关闭时错误消息逐字节不变 (保证现有 golden/精确字符串断言不被破坏);
// 开启后会把触发错误的值以 " (value: <val>)" 形式追加到消息后。
//
// 用法:
//
//	validate.Config(func(opt *validate.GlobalOption) { opt.ErrShowValue = true })
//	defer validate.ResetOption() // 必须复位, 避免污染其它测试/全局状态
func TestIssue_184_v2(t *testing.T) {
	t.Run("default off: message unchanged, no value appended", func(t *testing.T) {
		// 复用 issue #316 的精确消息场景做"默认逐字节不变"的回归保证
		dataFace, err := validate.FromJSONBytes([]byte(`{"value": 9223372036854775807}`))
		assert.NoErr(t, err)

		v := dataFace.Create()
		v.StringRule("value", "int")
		assert.False(t, v.Validate())

		msg := v.Errors.One()
		assert.Eq(t, "value value must be an integer", msg)
		assert.StrNotContains(t, msg, "value:")
		assert.StrNotContains(t, msg, "(value")
	})

	t.Run("opt-in on: int invalid value appended", func(t *testing.T) {
		validate.Config(func(opt *validate.GlobalOption) { opt.ErrShowValue = true })
		defer validate.ResetOption()

		// 用 Map 给一个明确的 int 非法值, 追加值即为该 int 本身
		v := validate.Map(map[string]any{"age": 200})
		v.StringRule("age", "max:100")
		assert.False(t, v.Validate())

		msg := v.Errors.FieldOne("age")
		assert.NotEq(t, "", msg)
		// 原消息仍在, 且尾部追加了触发值
		assert.StrContains(t, msg, "(value: 200)")
	})

	t.Run("opt-in on: string field invalid value appended", func(t *testing.T) {
		validate.Config(func(opt *validate.GlobalOption) { opt.ErrShowValue = true })
		defer validate.ResetOption()

		v := validate.Map(map[string]any{"name": "ab"})
		v.StringRule("name", "minLen:4")
		assert.False(t, v.Validate())

		msg := v.Errors.FieldOne("name")
		assert.NotEq(t, "", msg)
		assert.StrContains(t, msg, "(value: ab)")
	})

	t.Run("opt-in resets cleanly: after ResetOption message is unchanged again", func(t *testing.T) {
		validate.Config(func(opt *validate.GlobalOption) { opt.ErrShowValue = true })
		validate.ResetOption()

		v := validate.Map(map[string]any{"age": "abc"})
		v.StringRule("age", "int")
		assert.False(t, v.Validate())

		msg := v.Errors.FieldOne("age")
		assert.StrNotContains(t, msg, "(value:")
	})
}

// https://github.com/gookit/validate/issues/189
// 希望增加 StringMessage 方法
//
// 诉求: 像 StringRule 那样, 可以用 tag 写法(字符串)来声明字段的错误消息,
// 而不必使用 map 形态的 AddMessages/WithMessages。
//
// 实现: 新增 v.StringMessage(field, message) 与批量 v.StringMessages(MS)。
// 镜像 `message` struct tag 的字符串格式, 链式返回 *Validation。
//
// 字符串格式:
//   - 按校验器多条(以 "|" 分隔), 每段 "validator:message" -> 注册到 "field.validator" 键。
//     eg: "required:name is required|minLen:name is too short"
//   - 字段级单条(无 "validator:" 前缀) -> 注册到 "field" 键, 作为该字段任意校验器失败时的兜底消息。
//     eg: "name is invalid"
func TestIssue_189_v2(t *testing.T) {
	t.Run("per-validator messages: required uses its own custom msg", func(t *testing.T) {
		// empty value -> required fails (non-required validators skip empty),
		// so the "required" per-validator message is used.
		v := validate.Map(map[string]any{"name": ""})
		v.StringRule("name", "required|minLen:5")
		v.StringMessage("name", "required:name is required|minLen:name is too short")

		assert.False(t, v.Validate())
		assert.Eq(t, "name is required", v.Errors.Field("name")["required"])
	})

	t.Run("per-validator messages: minLen uses its own custom msg", func(t *testing.T) {
		// non-empty short value -> required passes, only minLen fails,
		// so the "minLen" per-validator message is used.
		v := validate.Map(map[string]any{"name": "ab"})
		v.StringRule("name", "required|minLen:5")
		v.StringMessage("name", "required:name is required|minLen:name is too short")

		assert.False(t, v.Validate())
		assert.Eq(t, "name is too short", v.Errors.Field("name")["minLen"])
	})

	t.Run("field-level fallback message", func(t *testing.T) {
		v := validate.Map(map[string]any{"name": "ab"})
		v.StringRule("name", "minLen:5")
		v.StringMessage("name", "name is invalid")

		assert.False(t, v.Validate())
		assert.Eq(t, "name is invalid", v.Errors.FieldOne("name"))
	})

	t.Run("trim whitespace around segments and parts", func(t *testing.T) {
		v := validate.Map(map[string]any{"name": ""})
		v.StringRule("name", "required")
		v.StringMessage("name", "  required : name is required  ")

		assert.False(t, v.Validate())
		assert.Eq(t, "name is required", v.Errors.FieldOne("name"))
	})

	t.Run("StringMessages batch", func(t *testing.T) {
		v := validate.Map(map[string]any{"name": "", "age": 5})
		v.StopOnError = false
		v.StringRules(validate.MS{
			"name": "required",
			"age":  "min:10",
		})
		v.StringMessages(validate.MS{
			"name": "required:name is required",
			"age":  "age is invalid", // field-level fallback
		})

		assert.False(t, v.Validate())
		assert.Eq(t, "name is required", v.Errors.FieldOne("name"))
		assert.Eq(t, "age is invalid", v.Errors.FieldOne("age"))
	})

	t.Run("chained with StringRule returns *Validation", func(t *testing.T) {
		v := validate.Map(map[string]any{"name": ""})
		// chainable: StringRule(...).StringMessage(...)
		v.StringRule("name", "required").
			StringMessage("name", "required:name is required")

		assert.False(t, v.Validate())
		assert.Eq(t, "name is required", v.Errors.FieldOne("name"))
	})
}
