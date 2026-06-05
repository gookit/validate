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

// https://github.com/gookit/validate/issues/292
// 希望支持完整逻辑条件(或/非)
//
// 诉求: 验证「字段是 IP 或 CIDR」, 但 `validate:"ip|CIDR"` 里 `|` 是规则分隔符(逻辑与,
// 两者都要满足), 无法表达「任一满足即可」。
//
// 实现: 新增组合校验器 `rule_one_of` (规则级"逻辑或"), 满足列出的任一子校验器即通过。
//   - 名称用 rule_one_of 而非 oneof: oneof 已是 enum 别名(语义=值∈集合), 会撞车。
//   - phase1 仅支持无参子校验器(ip/cidr/email/url/...); 带参子规则(min:5)留后续。
//   - 子项名支持别名(ip/IP/cidr/CIDR), 经 ValidatorName 解析。
//   - 未知子校验器名 → build/解析期 fail-fast(panic), 与现有未知 validator 行为一致。
//   - 错误消息: "{field} did not satisfy any of: %v"(%v 渲染子校验器名列表)。
func TestIssue_292_v2(t *testing.T) {
	t.Run("tag: ip hits -> pass", func(t *testing.T) {
		type Host struct {
			Addr string `validate:"rule_one_of:ip,cidr"`
		}
		v := validate.Struct(&Host{Addr: "1.2.3.4"})
		assert.True(t, v.Validate())
	})

	t.Run("tag: cidr hits -> pass", func(t *testing.T) {
		type Host struct {
			Addr string `validate:"rule_one_of:ip,cidr"`
		}
		v := validate.Struct(&Host{Addr: "10.0.0.0/8"})
		assert.True(t, v.Validate())
	})

	t.Run("tag: neither -> fail with message", func(t *testing.T) {
		type Host struct {
			Addr string `validate:"rule_one_of:ip,cidr"`
		}
		v := validate.Struct(&Host{Addr: "not-an-addr"})
		assert.False(t, v.Validate())
		assert.StrContains(t, v.Errors.FieldOne("Addr"), "did not satisfy any")
	})

	t.Run("programmatic StringRule: ip pass, invalid fail", func(t *testing.T) {
		v := validate.Map(map[string]any{"addr": "1.2.3.4"})
		v.StringRule("addr", "rule_one_of:ip,cidr")
		assert.True(t, v.Validate())

		v2 := validate.Map(map[string]any{"addr": "abc"})
		v2.StringRule("addr", "rule_one_of:ip,cidr")
		assert.False(t, v2.Validate())
		assert.StrContains(t, v2.Errors.FieldOne("addr"), "did not satisfy any")
	})

	t.Run("alias resolution: ip/IP/cidr/CIDR all hit", func(t *testing.T) {
		// uppercase aliases resolve via ValidatorName
		v := validate.Map(map[string]any{"addr": "1.2.3.4"})
		v.StringRule("addr", "rule_one_of:IP,CIDR")
		assert.True(t, v.Validate())

		v2 := validate.Map(map[string]any{"addr": "10.0.0.0/8"})
		v2.StringRule("addr", "rule_one_of:ip,cidr")
		assert.True(t, v2.Validate())
	})

	t.Run("combine with required", func(t *testing.T) {
		// empty -> rejected by required
		v := validate.Map(map[string]any{"addr": ""})
		v.StringRule("addr", "required|rule_one_of:ip,cidr")
		assert.False(t, v.Validate())

		// non-empty invalid -> rejected by rule_one_of
		v2 := validate.Map(map[string]any{"addr": "xxx"})
		v2.StringRule("addr", "required|rule_one_of:ip,cidr")
		assert.False(t, v2.Validate())
		assert.StrContains(t, v2.Errors.FieldOne("addr"), "did not satisfy any")

		// valid -> pass
		v3 := validate.Map(map[string]any{"addr": "1.2.3.4"})
		v3.StringRule("addr", "required|rule_one_of:ip,cidr")
		assert.True(t, v3.Validate())
	})

	t.Run("without required: empty value is skipped (SkipOnEmpty)", func(t *testing.T) {
		// rule_one_of is a non-required validator; empty value is skipped -> pass.
		v := validate.Map(map[string]any{"addr": ""})
		v.StringRule("addr", "rule_one_of:ip,cidr")
		assert.True(t, v.Validate())
	})

	t.Run("unknown sub-validator name -> panic (fail-fast)", func(t *testing.T) {
		// unknown name placed first so the loop reaches it regardless of value;
		// fail-fast panics rather than silently swallowing the typo.
		assert.Panics(t, func() {
			v := validate.Map(map[string]any{"addr": "1.2.3.4"})
			v.StringRule("addr", "rule_one_of:no_such_validator,ip")
			v.Validate()
		})
	})
}
