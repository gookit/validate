package validate

import (
	"reflect"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2/internal/fieldval"
)

func TestFieldCtx_methods(t *testing.T) {
	t.Run("string value", func(t *testing.T) {
		fv := fieldval.New("addr.city", "hello")
		fc := &fieldCtx{fv: fv, field: "addr.city", args: []any{"x", 2}}

		assert.Eq(t, "hello", fc.Value().String())
		assert.Eq(t, reflect.String, fc.Raw().Kind())
		assert.Eq(t, "addr.city", fc.FieldName())

		a0, ok0 := fc.Arg(0)
		assert.True(t, ok0)
		assert.Eq(t, "x", a0)

		a5, ok5 := fc.Arg(5)
		assert.False(t, ok5)
		assert.Nil(t, a5)

		assert.Eq(t, 2, len(fc.Args()))
	})

	t.Run("pointer value (de-pointered by Value)", func(t *testing.T) {
		p := new(int)
		*p = 7
		fv := fieldval.New("n", p)
		fc := &fieldCtx{fv: fv, field: "n"}

		assert.Eq(t, reflect.Int, fc.Value().Kind()) // RealV de-pointers
		assert.Eq(t, reflect.Ptr, fc.Raw().Kind())   // RV keeps the pointer
	})
}

func TestNewFuncMeta_style(t *testing.T) {
	t.Run("legacy func(val any) bool", func(t *testing.T) {
		fm := newFuncMeta("legacy", false, reflect.ValueOf(func(val any) bool { return true }))
		assert.Eq(t, styleLegacy, fm.style)
		assert.Nil(t, fm.fcFunc)
	})

	t.Run("fieldctx func(FieldCtx) bool", func(t *testing.T) {
		fm := newFuncMeta("fc", false, reflect.ValueOf(func(fc FieldCtx) bool { return true }))
		assert.Eq(t, styleFieldCtx, fm.style)
		assert.NotNil(t, fm.fcFunc)
	})
}

func TestFieldCtxValidator_pkg(t *testing.T) {
	// 用独特前缀避免污染全局并与其它用例撞名。
	AddValidator("isFooFC_r3b", func(fc FieldCtx) bool {
		s, _ := fc.Value().Interface().(string)
		return s == "foo"
	})

	t.Run("pass", func(t *testing.T) {
		v := New(map[string]any{"name": "foo"})
		v.AddRule("name", "isFooFC_r3b")
		assert.True(t, v.Validate())
	})

	t.Run("fail", func(t *testing.T) {
		v := New(map[string]any{"name": "bar"})
		v.AddRule("name", "isFooFC_r3b")
		assert.False(t, v.Validate())
	})
}

func TestFieldCtxValidator_instance(t *testing.T) {
	// 实例级注册,无全局污染。带参规则: args 经 fc.Arg/Args 取,且应为原始未转换值。
	newV := func(age any) *Validation {
		v := New(map[string]any{"age": age})
		v.AddValidator("inRangeFC", func(fc FieldCtx) bool {
			// 确认拿到原始 args(字符串 "1"/"100"),未被 convertArgsType 转换。
			a0, ok0 := fc.Arg(0)
			a1, ok1 := fc.Arg(1)
			if !ok0 || !ok1 {
				return false
			}
			if a0 != "1" || a1 != "100" {
				return false
			}
			if got := fc.Args(); len(got) != 2 {
				return false
			}
			if fc.FieldName() != "age" {
				return false
			}
			n, _ := fc.Value().Interface().(int)
			lo, hi := 1, 100
			return n >= lo && n <= hi
		})
		v.AddRule("age", "inRangeFC", "1", "100")
		return v
	}

	t.Run("pass", func(t *testing.T) {
		assert.True(t, newV(50).Validate())
	})
	t.Run("fail-out-of-range", func(t *testing.T) {
		assert.False(t, newV(200).Validate())
	})
}

func TestFieldCtxValidator_coexist(t *testing.T) {
	// 同一 Validation 同时注册 legacy func(val any)bool 与 func(FieldCtx)bool, 两者都生效。
	v := New(map[string]any{"a": "legacy-ok", "b": "fc-ok"})
	v.AddValidator("legacyChk", func(val any) bool {
		return val == "legacy-ok"
	})
	v.AddValidator("fcChk", func(fc FieldCtx) bool {
		s, _ := fc.Value().Interface().(string)
		return s == "fc-ok"
	})
	v.AddRule("a", "legacyChk")
	v.AddRule("b", "fcChk")
	assert.True(t, v.Validate())

	// 反例: legacy 路径仍按旧行为判败。
	v2 := New(map[string]any{"a": "wrong", "b": "fc-ok"})
	v2.AddValidator("legacyChk", func(val any) bool {
		return val == "legacy-ok"
	})
	v2.AddValidator("fcChk", func(fc FieldCtx) bool {
		s, _ := fc.Value().Interface().(string)
		return s == "fc-ok"
	})
	v2.AddRule("a", "legacyChk")
	v2.AddRule("b", "fcChk")
	assert.False(t, v2.Validate())
}

func TestFieldCtxValidator_wildcardSlice(t *testing.T) {
	// fieldctx 作用在 ".*" 切片字段: 每个元素都走到 fieldctx 调用。
	seen := 0
	v := New(map[string]any{
		"tags": []string{"go", "rust", "zig"},
	})
	v.AddValidator("nonEmptyFC", func(fc FieldCtx) bool {
		seen++
		s, _ := fc.Value().Interface().(string)
		return s != ""
	})
	v.AddRule("tags.*", "nonEmptyFC")
	assert.True(t, v.Validate())
	assert.Eq(t, 3, seen) // 每个元素都被校验到

	// 含空元素 → 失败。
	v2 := New(map[string]any{
		"tags": []string{"go", "", "zig"},
	})
	v2.AddValidator("nonEmptyFC", func(fc FieldCtx) bool {
		s, _ := fc.Value().Interface().(string)
		return s != ""
	})
	v2.AddRule("tags.*", "nonEmptyFC")
	assert.False(t, v2.Validate())
}
