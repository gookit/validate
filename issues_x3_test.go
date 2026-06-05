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
