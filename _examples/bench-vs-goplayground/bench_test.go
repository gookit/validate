// Package main 仅承载 vs go-playground/validator 的对照 benchmark。
//
// 设计目标：apples-to-apples —— 对同一组等价场景，分别用 gookit/validate 与
// go-playground/validator 跑相同语义的规则，只测“校验调用”本身。
//
// 注意（公平性）：两库特性/语义不同（gookit 集 校验+过滤+绑定，每次 validate.Struct
// 会构建一个 Validation 实例并解析 tag；go-playground 专注校验，结构体元信息进
// 内部缓存）。两边都尽量做到“循环外建好可复用的东西、循环内只跑校验”，但 gookit
// 当前 public API 每轮仍需 Struct() 构建实例——这正是其真实使用成本，如实测量。
package main

import (
	"testing"

	playground "github.com/go-playground/validator/v10"
	gookit "github.com/gookit/validate/v2"
)

/*************************************************************
 * 场景 1：扁平 struct —— 成功路径（合法数据，全通过）
 *************************************************************/

// gookitFlat 使用 gookit 规则语法：`|` 分隔规则，`:` 传参。
type gookitFlat struct {
	Name  string `validate:"required|min_len:3|max_len:20"`
	Email string `validate:"required|email"`
	Age   int    `validate:"required|min:1|max:120"`
}

// playgroundFlat 使用 go-playground 规则语法：`,` 分隔规则，`=` 传参。
// 字段数与规则语义与 gookitFlat 对齐：required、长度 3~20、email、数值 1~120。
type playgroundFlat struct {
	Name  string `validate:"required,min=3,max=20"`
	Email string `validate:"required,email"`
	Age   int    `validate:"required,gte=1,lte=120"`
}

func validFlatGookit() gookitFlat {
	return gookitFlat{Name: "inhere", Email: "john@example.com", Age: 30}
}

func validFlatPlayground() playgroundFlat {
	return playgroundFlat{Name: "inhere", Email: "john@example.com", Age: 30}
}

func BenchmarkGookitFlatValid(b *testing.B) {
	data := validFlatGookit()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := gookit.Struct(&data)
		if !v.Validate() {
			b.Fatalf("expected valid, got: %v", v.Errors)
		}
	}
}

func BenchmarkPlaygroundFlatValid(b *testing.B) {
	validate := playground.New()
	data := validFlatPlayground()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := validate.Struct(&data); err != nil {
			b.Fatalf("expected valid, got: %v", err)
		}
	}
}

// Check 是 gookit 的池化无状态入口(对标 playground 复用单实例的方式):
// 内部用包级 sync.Pool 复用 *Validation,返回与实例解耦的 *ValidResult。
func BenchmarkGookitCheckValid(b *testing.B) {
	data := validFlatGookit()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := gookit.Check(&data)
		if r.Fail() {
			b.Fatalf("expected valid, got: %v", r.Errors)
		}
	}
}

func BenchmarkGookitCheckErrValid(b *testing.B) {
	data := validFlatGookit()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if gookit.CheckErr(&data) != nil {
			b.Fatalf("expected valid")
		}
	}
}

/*************************************************************
 * 场景 2：扁平 struct —— 失败路径（含非法字段，触发错误）
 *
 * 同一份非法数据：Name 太短、Email 非法、Age 越界。
 *************************************************************/

func invalidFlatGookit() gookitFlat {
	return gookitFlat{Name: "x", Email: "not-an-email", Age: 999}
}

func invalidFlatPlayground() playgroundFlat {
	return playgroundFlat{Name: "x", Email: "not-an-email", Age: 999}
}

func BenchmarkGookitFlatInvalid(b *testing.B) {
	data := invalidFlatGookit()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := gookit.Struct(&data)
		if v.Validate() {
			b.Fatal("expected invalid, got pass")
		}
	}
}

func BenchmarkPlaygroundFlatInvalid(b *testing.B) {
	validate := playground.New()
	data := invalidFlatPlayground()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := validate.Struct(&data); err == nil {
			b.Fatal("expected invalid, got pass")
		}
	}
}

/*************************************************************
 * 场景 3：嵌套 struct（成功路径）
 *
 * 级联语义不同（见 README）：
 *   - gookit：子 struct 字段带 validate tag（空 tag 也算）即级联子字段规则；
 *     这里给 Profile 字段加 `validate:"required"` 触发级联。
 *   - go-playground：嵌套 struct 默认会递归校验其字段，无需特殊 tag；
 *     `dive` 用于 slice/map 元素，这里是单个嵌套 struct 故不需要 dive。
 *************************************************************/

type gookitProfile struct {
	City string `validate:"required|min_len:2"`
	Zip  string `validate:"required|min_len:3|max_len:10"`
}

type gookitNested struct {
	Name    string        `validate:"required|min_len:3"`
	Profile gookitProfile `validate:"required"`
}

type playgroundProfile struct {
	City string `validate:"required,min=2"`
	Zip  string `validate:"required,min=3,max=10"`
}

type playgroundNested struct {
	Name    string            `validate:"required,min=3"`
	Profile playgroundProfile `validate:"required"`
}

func validNestedGookit() gookitNested {
	return gookitNested{
		Name:    "inhere",
		Profile: gookitProfile{City: "Beijing", Zip: "100000"},
	}
}

func validNestedPlayground() playgroundNested {
	return playgroundNested{
		Name:    "inhere",
		Profile: playgroundProfile{City: "Beijing", Zip: "100000"},
	}
}

func BenchmarkGookitNestedValid(b *testing.B) {
	data := validNestedGookit()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := gookit.Struct(&data)
		if !v.Validate() {
			b.Fatalf("expected valid, got: %v", v.Errors)
		}
	}
}

func BenchmarkPlaygroundNestedValid(b *testing.B) {
	validate := playground.New()
	data := validNestedPlayground()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := validate.Struct(&data); err != nil {
			b.Fatalf("expected valid, got: %v", err)
		}
	}
}
