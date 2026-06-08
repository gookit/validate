package validate

import (
	"testing"

	"github.com/gookit/goutil/x/assert"

	"github.com/gookit/validate/v2/internal/fieldval"
	ivalidators "github.com/gookit/validate/v2/internal/validators"
)

// R2.5a 等价契约测试:internal RV 版 ≡ public 作用于 carrier.RealV().Interface()。
// reflect.Call 路径在调 public 前会做一次 RealV 预解引用,故 public 实际收到的是
// vfv.RealV().Interface()(vfv==nil 时等价 fieldval.New(field,val).RealV().Interface())。

// 命名类型(动态类型断言 vs Kind 的关键用例)
type myBool bool
type myInt int
type myUint uint
type myStr string

// realVAny 复现 reflect.Call 在调 public 前的预解引用:返回 New("",v).RealV().Interface()。
func realVAny(v any) any {
	return fieldval.New("", v).RealV().Interface()
}

// r25Matrix 覆盖各类输入值。**T 双指针、nil-ptr、命名类型、NilObject 都在内。
func r25Matrix() []any {
	pi := 7
	ppi := &pi
	var nilIntP *int
	pb := true
	ps := "hi"
	sl := []int{1, 2, 3}
	psl := &sl
	mp := map[string]int{"a": 1}
	pmp := &mp
	arr := [3]int{1, 2, 3}

	return []any{
		// bool
		true, false, myBool(true), myBool(false),
		&pb,
		// ints
		0, 1, -1,
		int8(1), int8(-1), int16(2), int32(3), int64(4), int64(-4),
		myInt(5), myInt(-5),
		// uints
		uint(1), uint8(2), uint16(3), uint32(4), uint64(5), myUint(6),
		// floats
		1.5, float32(2.5),
		// strings
		"true", "1", "abc", "", "-5", "18446744073709551615", // last > int64 max, uint range edge
		myStr("true"), myStr("abc"),
		ps, &ps,
		// slices / arrays / maps
		sl, psl, arr, mp, pmp,
		[]string{"x"}, []any{},
		// pointers
		ppi,        // *int
		&ppi,       // **int 双指针
		nilIntP,    // (*int)(nil)
		nil,        // untyped nil
		NilObject{}, // 内部 NilObject
	}
}

func TestR25a_IsBool_equiv(t *testing.T) {
	for _, v := range r25Matrix() {
		got := ivalidators.IsBool(fieldval.New("", v))
		want := IsBool(realVAny(v))
		assert.Require(t, assert.Eq(t, want, got, "IsBool mismatch for %#v", v))
	}
}

func TestR25a_IsUint_equiv(t *testing.T) {
	for _, v := range r25Matrix() {
		got := ivalidators.IsUint(fieldval.New("", v))
		want := IsUint(realVAny(v))
		assert.Require(t, assert.Eq(t, want, got, "IsUint mismatch for %#v", v))
	}
}

func TestR25a_IsArray_equiv(t *testing.T) {
	for _, v := range r25Matrix() {
		t.Run("default", func(t *testing.T) {
			got := ivalidators.IsArray(fieldval.New("", v))
			want := IsArray(realVAny(v))
			assert.Require(t, assert.Eq(t, want, got, "IsArray(default) mismatch for %#v", v))
		})
		t.Run("strict-true", func(t *testing.T) {
			got := ivalidators.IsArray(fieldval.New("", v), true)
			want := IsArray(realVAny(v), true)
			assert.Require(t, assert.Eq(t, want, got, "IsArray(strict) mismatch for %#v", v))
		})
		t.Run("strict-false", func(t *testing.T) {
			got := ivalidators.IsArray(fieldval.New("", v), false)
			want := IsArray(realVAny(v), false)
			assert.Require(t, assert.Eq(t, want, got, "IsArray(strict=false) mismatch for %#v", v))
		})
	}
}

func TestR25a_IsMap_equiv(t *testing.T) {
	for _, v := range r25Matrix() {
		got := ivalidators.IsMap(fieldval.New("", v))
		want := IsMap(realVAny(v))
		assert.Require(t, assert.Eq(t, want, got, "IsMap mismatch for %#v", v))
	}
}

// 端到端: 经 AddRule 走 callValidator 的新 switch case, 验证分发正确。
func TestR25a_switch_dispatch(t *testing.T) {
	t.Run("isBool", func(t *testing.T) {
		v := New(map[string]any{"ok": true, "bad": []int{1}})
		v.StringRule("ok", "bool")
		assert.True(t, v.Validate())

		v2 := New(map[string]any{"bad": []int{1}})
		v2.StringRule("bad", "bool")
		assert.False(t, v2.Validate())
	})

	t.Run("isUint", func(t *testing.T) {
		v := New(map[string]any{"n": uint(3)})
		v.StringRule("n", "uint")
		assert.True(t, v.Validate())

		v2 := New(map[string]any{"n": -3})
		v2.StringRule("n", "uint")
		assert.False(t, v2.Validate())
	})

	t.Run("isArray", func(t *testing.T) {
		v := New(map[string]any{"a": []int{1, 2}})
		v.StringRule("a", "isArray")
		assert.True(t, v.Validate())

		v2 := New(map[string]any{"a": "nope"})
		v2.StringRule("a", "isArray")
		assert.False(t, v2.Validate())
	})

	t.Run("isMap", func(t *testing.T) {
		v := New(map[string]any{"m": map[string]int{"x": 1}})
		v.StringRule("m", "isMap")
		assert.True(t, v.Validate())

		v2 := New(map[string]any{"m": []int{1}})
		v2.StringRule("m", "isMap")
		assert.False(t, v2.Validate())
	})
}

// ===== R2.5b: Contains / NotContains 提升进 switch(不搬 internal) =====

// 等价契约: switch 表达式 Contains(c.RealV().Interface(), sub) ≡ 原 default 路径
// reflect.Call 调 includeElement(vfv.RealV(), sub)。reflect.Call 路径里 public Contains
// 收到的就是 vfv.RealV().Interface(),故二者按定义恒等;此处断言 switch 表达式与
// 「直接 Contains(realVAny(v), sub)」一致(覆盖 []T / *[]T 指针容器 / map / string / nil)。
func TestR25b_Contains_equiv(t *testing.T) {
	sl := []string{"go", "rust"}
	psl := &sl
	mp := map[string]int{"a": 1, "b": 2}
	cases := []struct {
		v   any
		sub any
	}{
		{[]string{"go", "rust"}, "go"},
		{[]string{"go", "rust"}, "x"},
		{psl, "rust"},        // *[]string 指针容器, 验证 RealV 解引用生效
		{psl, "x"},           // 指针容器未命中
		{mp, "a"},            // map key 命中
		{mp, "zzz"},          // map key 未命中
		{"hello world", "wor"}, // string 包含
		{"hello", "zzz"},     // string 不包含
		{nil, "x"},           // nil 容器
	}
	for _, c := range cases {
		// switch 表达式
		got := Contains(fieldval.New("", c.v).RealV().Interface(), c.sub)
		// 等价参考(同一表达式 = reflect.Call 路径语义)
		want := Contains(realVAny(c.v), c.sub)
		assert.Require(t, assert.Eq(t, want, got, "Contains mismatch v=%#v sub=%#v", c.v, c.sub))

		gotN := NotContains(fieldval.New("", c.v).RealV().Interface(), c.sub)
		wantN := NotContains(realVAny(c.v), c.sub)
		assert.Require(t, assert.Eq(t, wantN, gotN, "NotContains mismatch v=%#v sub=%#v", c.v, c.sub))
	}
}

// 端到端: 经 AddRule 走 callValidator 的 contains/notContains 新 case。
func TestR25b_switch_dispatch(t *testing.T) {
	t.Run("contains-slice", func(t *testing.T) {
		v := New(map[string]any{"tags": []string{"go", "rust"}})
		v.AddRule("tags", "contains", "go")
		assert.True(t, v.Validate())

		v2 := New(map[string]any{"tags": []string{"go", "rust"}})
		v2.AddRule("tags", "contains", "x")
		assert.False(t, v2.Validate())
	})

	t.Run("contains-string", func(t *testing.T) {
		v := New(map[string]any{"s": "hello world"})
		v.AddRule("s", "contains", "wor")
		assert.True(t, v.Validate())
	})

	t.Run("contains-map-key", func(t *testing.T) {
		v := New(map[string]any{"m": map[string]int{"a": 1, "b": 2}})
		v.AddRule("m", "contains", "a")
		assert.True(t, v.Validate())
	})

	t.Run("notContains", func(t *testing.T) {
		v := New(map[string]any{"tags": []string{"go", "rust"}})
		v.AddRule("tags", "notContains", "x")
		assert.True(t, v.Validate())

		v2 := New(map[string]any{"tags": []string{"go", "rust"}})
		v2.AddRule("tags", "notContains", "go")
		assert.False(t, v2.Validate())
	})
}
