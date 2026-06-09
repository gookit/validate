package validate

import (
	"sync"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

type checkUser struct {
	Name  string `validate:"required|min_len:3" json:"name"`
	Email string `validate:"required|email" json:"email"`
	Age   int    `validate:"required|min:1|max:120" json:"age"`
}

func validCheckUser() *checkUser {
	return &checkUser{Name: "inhere", Email: "test@example.com", Age: 30}
}

func invalidCheckUser() *checkUser {
	return &checkUser{Name: "x", Email: "not-an-email", Age: 999}
}

// Check on valid struct: ok face + safe data + bind.
func TestCheck_valid(t *testing.T) {
	r := Check(validCheckUser())

	assert.True(t, r.IsOK())
	assert.False(t, r.Fail())
	assert.NoErr(t, r.Err())
	assert.True(t, r.Errors.Empty())

	// safe data keyed by struct field name
	assert.NotEmpty(t, r.SafeData())
	assert.Eq(t, "inhere", r.SafeVal("Name"))
	val, ok := r.Safe("Age")
	assert.True(t, ok)
	assert.Eq(t, 30, val)

	// bind back to a struct (json keys, case-insensitive match)
	var out checkUser
	assert.NoErr(t, r.BindStruct(&out))
	assert.Eq(t, "inhere", out.Name)
	assert.Eq(t, 30, out.Age)
}

// Check on invalid struct: fail face + empty safe data.
func TestCheck_invalid(t *testing.T) {
	r := Check(invalidCheckUser())

	assert.False(t, r.IsOK())
	assert.True(t, r.Fail())
	assert.Err(t, r.Err())
	assert.False(t, r.Errors.Empty())

	// safe data cleared on error
	assert.Empty(t, r.SafeData())

	// BindStruct is a no-op (no safe data) and returns nil
	var out checkUser
	assert.NoErr(t, r.BindStruct(&out))
	assert.Eq(t, "", out.Name)
}

// ValidateR is the primitive: works on any configured instance (struct).
func TestValidateR_struct(t *testing.T) {
	r := Struct(validCheckUser()).ValidateR()
	assert.True(t, r.IsOK())
	assert.Eq(t, "inhere", r.SafeVal("Name"))

	r = Struct(invalidCheckUser()).ValidateR()
	assert.True(t, r.Fail())
}

// ValidateR on a map + programmatic rules (the documented map path).
func TestValidateR_map(t *testing.T) {
	v := Map(map[string]any{"age": 100, "name": "inhere"})
	v.StringRules(MS{
		"name": "required|string",
		"age":  "required|int|min:1",
	})

	r := v.ValidateR()
	assert.True(t, r.IsOK())
	assert.Eq(t, 100, r.SafeVal("age"))
	assert.Eq(t, "inhere", r.SafeVal("name"))
}

// ValidateR with a scene arg routes through Validate(scene...).
func TestValidateR_scene(t *testing.T) {
	v := Map(map[string]any{"name": "ab"})
	v.StringRule("name", "required|minLen:5")
	v.WithScenes(SValues{"s1": []string{"name"}})

	// scene s1 checks name -> fails
	r := v.ValidateR("s1")
	assert.True(t, r.Fail())
}

// The returned result is a snapshot, independent of later pooled reuse.
func TestCheck_resultIndependentAfterReuse(t *testing.T) {
	r1 := Check(validCheckUser())
	assert.True(t, r1.IsOK())
	name := r1.SafeVal("Name")

	// hammer the pool with other validations (valid + invalid mixed)
	for i := 0; i < 50; i++ {
		_ = Check(invalidCheckUser())
		_ = Check(&checkUser{Name: "other", Email: "o@e.com", Age: 10})
	}

	// r1 must be unchanged by the reuse
	assert.True(t, r1.IsOK())
	assert.Eq(t, name, r1.SafeVal("Name"))
	assert.Eq(t, "inhere", r1.SafeVal("Name"))
}

// Repeated Check calls reuse pooled instances but yield independent, correct
// results with no cross-contamination.
func TestCheck_poolReuse(t *testing.T) {
	for i := 0; i < 100; i++ {
		rok := Check(validCheckUser())
		assert.True(t, rok.IsOK())
		assert.True(t, rok.Errors.Empty())

		rbad := Check(invalidCheckUser())
		assert.True(t, rbad.Fail())
		assert.False(t, rbad.Errors.Empty())
	}
}

type checkAddr struct {
	City string `validate:"required|min_len:2" json:"city"`
	Zip  string `validate:"required" json:"zip"`
}

// S3: the pooled StructData (v.sd) is reused across calls — including across
// DIFFERENT struct types. Verify no field-name bleed / source leak between types.
func TestCheck_poolReuse_mixedTypes(t *testing.T) {
	for i := 0; i < 50; i++ {
		ru := Check(validCheckUser()) // type checkUser
		assert.True(t, ru.IsOK())
		assert.Eq(t, "inhere", ru.SafeVal("Name"))

		ra := Check(&checkAddr{City: "NYC", Zip: "10001"}) // different type, reused pool slot
		assert.True(t, ra.IsOK())
		assert.Eq(t, "NYC", ra.SafeVal("City"))
		// checkUser's fields must NOT leak into the addr result
		_, hasName := ra.Safe("Name")
		assert.False(t, hasName)

		rbad := Check(&checkAddr{City: "x"}) // City too short + Zip missing
		assert.True(t, rbad.Fail())
	}
}

// Concurrency: Check of MIXED types from many goroutines must be race-clean and
// correct (each goroutine gets its own pooled instance + StructData).
func TestCheck_concurrent_mixedTypes(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if r := Check(validCheckUser()); !r.IsOK() || r.SafeVal("Name") != "inhere" {
					t.Errorf("user: unexpected result: ok=%v name=%v", r.IsOK(), r.SafeVal("Name"))
				}
				if r := Check(&checkAddr{City: "NYC", Zip: "10001"}); !r.IsOK() || r.SafeVal("City") != "NYC" {
					t.Errorf("addr: unexpected result: ok=%v city=%v", r.IsOK(), r.SafeVal("City"))
				}
			}
		}()
	}
	wg.Wait()
}

// Concurrency: Check must be safe from many goroutines (run with -race).
func TestCheck_concurrent(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if r := Check(validCheckUser()); !r.IsOK() {
					t.Errorf("expected ok, got errors: %v", r.Errors)
				}
				if r := Check(invalidCheckUser()); !r.Fail() {
					t.Errorf("expected fail, got ok")
				}
			}
		}()
	}
	wg.Wait()
}

// Filtered data is carried over into the result.
func TestValidResult_filtered(t *testing.T) {
	v := Map(map[string]any{"age": "100", "name": "inhere"})
	v.FilterRule("age", "int")
	v.StringRules(MS{
		"name": "required|string",
		"age":  "required|int|min:1",
	})

	r := v.ValidateR()
	assert.True(t, r.IsOK())
	assert.Eq(t, 100, r.Filtered("age"))
	assert.NotEmpty(t, r.FilteredData())
}

func TestCheckErr_basic(t *testing.T) {
	assert.NoErr(t, CheckErr(validCheckUser()))
	assert.Err(t, CheckErr(invalidCheckUser()))
}

// CheckErr 用过的池实例,后续 Check 必须仍能正常收 safeData(resetForReuse 清 skipCollect)
func TestCheckErr_poolNoContaminationWithCheck(t *testing.T) {
	for i := 0; i < 30; i++ {
		assert.NoErr(t, CheckErr(validCheckUser()))
		r := Check(validCheckUser())
		assert.True(t, r.IsOK())
		assert.Eq(t, "inhere", r.SafeVal("Name")) // safeData 仍被收集
	}
}

// =============================================================================
// CheckErr P2 测试矩阵 (docs/perf/checkerr-impl-plan.md §6)
//
// 核心策略: differential testing —— 对多样 struct 断言
//
//	(CheckErr(s) == nil) == Check(s).IsOK()
//
// 直接证明 CheckErr 与 Check 过/败一致。差异化用例 **务必每次用新建结构体**:
// struct 源 UpdateSource=true,默认值/过滤值会写回源,复用同一指针多次校验会因源
// 被改写而行为不同(这是 gookit 既有语义,与 CheckErr 无关)。
// =============================================================================

// 跨字段: Confirm 必须等于 Pwd。Confirm 带 filter:"trim",验证"过滤值经 1 槽/源
// 写回正确传递给跨字段校验器"(计划 §4 写回论证)。
type ceCrossField struct {
	Pwd     string `validate:"required" json:"pwd"`
	Confirm string `validate:"required|eq_field:Pwd" filter:"trim" json:"confirm"`
}

// filter + 多规则同字段: Name 带 filter:"trim|lower" + 3 条 validate 规则。
type ceFilterMulti struct {
	Name string `validate:"required|min_len:3|max_len:10" filter:"trim|lower" json:"name"`
	Tag  string `validate:"required|in:a,b,c" filter:"trim|lower" json:"tag"`
}

// 默认值: 零值字段 + default tag (struct-only,经 tag 解析 SetDefValue)。
type ceDefault struct {
	Name string `validate:"required|min_len:3|default:tom" json:"name"`
	Age  int    `validate:"uint|max:150|default:23" json:"age"`
}

// required_with: B 在 A 存在时必填。A 带 filter:"trim"。
type ceReqWith struct {
	A string `validate:"required" filter:"trim" json:"a"`
	B string `validate:"required_with:A" json:"b"`
}

// gt_field: Max 必须大于 Min。
type ceGtField struct {
	Min int `validate:"required|int" json:"min"`
	Max int `validate:"required|gt_field:Min" json:"max"`
}

// differentialCases 返回一组工厂函数(每次调用产新实例,避免源写回串扰)。
func differentialCases() []func() any {
	return []func() any{
		// --- 合法 ---
		func() any { return validCheckUser() },
		func() any { return &checkAddr{City: "NYC", Zip: "10001"} },
		func() any { return &ceCrossField{Pwd: "abc123", Confirm: " abc123 "} }, // trim 后相等
		func() any { return &ceCrossField{Pwd: "abc123", Confirm: "abc123"} },
		func() any { return &ceFilterMulti{Name: " Inhere ", Tag: " A "} }, // trim|lower 后合法
		func() any { return &ceDefault{} },                                // 全默认 -> 合法
		func() any { return &ceDefault{Name: "alice", Age: 40} },
		func() any { return &ceReqWith{A: "x", B: "y"} },
		func() any { return &ceReqWith{} }, // A 为空 -> required_with 不触发,但 A required 失败
		func() any { return &ceGtField{Min: 1, Max: 9} },
		// --- 非法 ---
		func() any { return invalidCheckUser() },
		func() any { return &checkAddr{City: "x"} },                            // City 太短 + Zip 缺失
		func() any { return &ceCrossField{Pwd: "abc123", Confirm: "xyz"} },     // 不等
		func() any { return &ceFilterMulti{Name: " ab ", Tag: " z "} },         // name 太短 + tag 不在集合
		func() any { return &ceFilterMulti{Name: " this-is-way-too-long " } },  // name 超长 + tag 缺失
		func() any { return &ceDefault{Name: "x", Age: 5} },                    // name 太短
		func() any { return &ceDefault{Age: 999} },                            // age 超 max
		func() any { return &ceReqWith{A: "x"} },                              // A 存在 -> B 必填,B 空 -> 失败
		func() any { return &ceGtField{Min: 9, Max: 1} },                       // max 不大于 min
		func() any { return &ceGtField{Min: 5, Max: 5} },                       // 相等 -> 不满足 gt
	}
}

// TestCheckErr_vsCheck_differential: 核心差异化断言。
func TestCheckErr_vsCheck_differential(t *testing.T) {
	for i, mk := range differentialCases() {
		ceErr := CheckErr(mk()) // 新实例
		ckRes := Check(mk())    // 新实例
		assert.Eq(t, ckRes.IsOK(), ceErr == nil,
			"case %d: CheckErr-ok=%v (err=%v) but Check.IsOK=%v", i, ceErr == nil, ceErr, ckRes.IsOK())
	}
}

// TestCheckErr_multiRuleSameField: 同字段 3+ 规则,合法->nil;让其中一条失败->err。
func TestCheckErr_multiRuleSameField(t *testing.T) {
	t.Run("all-pass", func(t *testing.T) {
		assert.NoErr(t, CheckErr(&ceFilterMulti{Name: "inhere", Tag: "a"}))
	})
	t.Run("min_len-fail", func(t *testing.T) {
		assert.Err(t, CheckErr(&ceFilterMulti{Name: "ab", Tag: "a"}))
	})
	t.Run("max_len-fail", func(t *testing.T) {
		assert.Err(t, CheckErr(&ceFilterMulti{Name: "this-name-is-too-long", Tag: "a"}))
	})
	t.Run("required-fail", func(t *testing.T) {
		assert.Err(t, CheckErr(&ceFilterMulti{Tag: "a"}))
	})
}

// TestCheckErr_filterMultiRule: filter + 多规则同字段,与 Check 一致(过滤值经 1 槽/源写回)。
func TestCheckErr_filterMultiRule(t *testing.T) {
	t.Run("filtered-becomes-valid", func(t *testing.T) {
		// " Inhere " --trim|lower--> "inhere" (len 6, 合法)
		assert.NoErr(t, CheckErr(&ceFilterMulti{Name: " Inhere ", Tag: " A "}))
		// Check 同样应通过
		assert.True(t, Check(&ceFilterMulti{Name: " Inhere ", Tag: " A "}).IsOK())
	})
	t.Run("filtered-still-invalid", func(t *testing.T) {
		// " ab " --trim|lower--> "ab" (len 2, min_len:3 失败)
		assert.Err(t, CheckErr(&ceFilterMulti{Name: " ab ", Tag: "a"}))
		assert.False(t, Check(&ceFilterMulti{Name: " ab ", Tag: "a"}).IsOK())
	})
}

// TestCheckErr_crossField_withFilter: 跨字段校验器引用带 filter 的字段(计划 §4 写回论证)。
func TestCheckErr_crossField_withFilter(t *testing.T) {
	t.Run("eq_field-after-trim-equal", func(t *testing.T) {
		// Confirm " abc123 " --trim--> "abc123" == Pwd "abc123" -> 通过
		assert.NoErr(t, CheckErr(&ceCrossField{Pwd: "abc123", Confirm: " abc123 "}))
		assert.True(t, Check(&ceCrossField{Pwd: "abc123", Confirm: " abc123 "}).IsOK())
	})
	t.Run("eq_field-not-equal", func(t *testing.T) {
		assert.Err(t, CheckErr(&ceCrossField{Pwd: "abc123", Confirm: "different"}))
		assert.False(t, Check(&ceCrossField{Pwd: "abc123", Confirm: "different"}).IsOK())
	})
	t.Run("required_with-and-gt_field", func(t *testing.T) {
		assert.NoErr(t, CheckErr(&ceReqWith{A: " x ", B: "y"}))
		assert.Err(t, CheckErr(&ceReqWith{A: "x"})) // B 必填但空
		assert.NoErr(t, CheckErr(&ceGtField{Min: 1, Max: 2}))
		assert.Err(t, CheckErr(&ceGtField{Min: 2, Max: 1}))
	})
}

// TestCheckErr_default_vsCheck: zero 值 + default tag(struct-only,经 tag 解析)。
// 说明: CheckErr 不暴露 *Validation,无法经 SetDefValue 手动设默认值;但 default
// tag 路径(parseRulesFromTag -> SetDefValue)对 struct 源生效,故此处可覆盖。
func TestCheckErr_default_vsCheck(t *testing.T) {
	t.Run("zero-fields-use-default", func(t *testing.T) {
		// Name 默认 "tom"(len 3 满足 min_len:3),Age 默认 23 -> 合法
		assert.NoErr(t, CheckErr(&ceDefault{}))
		assert.True(t, Check(&ceDefault{}).IsOK())
	})
	t.Run("explicit-invalid-overrides-default", func(t *testing.T) {
		// 显式 Name "x"(非零)-> 不走默认 -> min_len:3 失败
		assert.Err(t, CheckErr(&ceDefault{Name: "x"}))
		assert.False(t, Check(&ceDefault{Name: "x"}).IsOK())
	})
}

// TestCheckErr_skipEmpty_semantics: CheckErr 不暴露 v 来设 SkipOnEmpty(实例级开关),
// 故通过 differential(Check 也用默认 gOpt)+ tag 级 optional 语义覆盖等价场景。
// uint 校验器对零值字段默认 skipEmpty,这里断言 CheckErr 与 Check 对"空可选字段"一致。
func TestCheckErr_skipEmpty_semantics(t *testing.T) {
	type ceOptional struct {
		Name string `validate:"required" json:"name"`
		Note string `validate:"min_len:5" json:"note"` // 空时跳过(skipEmpty 默认)
	}
	t.Run("empty-optional-skipped", func(t *testing.T) {
		assert.NoErr(t, CheckErr(&ceOptional{Name: "x"})) // Note 空 -> 跳过
		assert.True(t, Check(&ceOptional{Name: "x"}).IsOK())
	})
	t.Run("nonempty-optional-validated", func(t *testing.T) {
		assert.Err(t, CheckErr(&ceOptional{Name: "x", Note: "ab"})) // Note 非空且太短
		assert.False(t, Check(&ceOptional{Name: "x", Note: "ab"}).IsOK())
	})
}

// TestCheckErr_poolReuse_alternatingTypes: 池复用 CheckErr->CheckErr 异类型交替,
// 合法/非法混合,断言各自结果正确(1 槽不串味)。
func TestCheckErr_poolReuse_alternatingTypes(t *testing.T) {
	for i := 0; i < 100; i++ {
		assert.NoErr(t, CheckErr(validCheckUser()))                          // typeA 合法
		assert.Err(t, CheckErr(&checkAddr{City: "x"}))                       // typeB 非法
		assert.NoErr(t, CheckErr(&ceFilterMulti{Name: "inhere", Tag: "b"})) // typeC 合法
		assert.Err(t, CheckErr(invalidCheckUser()))                          // typeA 非法
		assert.NoErr(t, CheckErr(&checkAddr{City: "NYC", Zip: "10001"}))    // typeB 合法
	}
}

// TestCheckErr_concurrent_mixedWithCheck: 多 goroutine 同时跑 Check 和 CheckErr
// (同/异类型),断言结果正确、无竞争(-race)。
func TestCheckErr_concurrent_mixedWithCheck(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if CheckErr(validCheckUser()) != nil {
					t.Errorf("CheckErr(valid user): expected nil")
				}
				if CheckErr(invalidCheckUser()) == nil {
					t.Errorf("CheckErr(invalid user): expected err")
				}
				if r := Check(validCheckUser()); !r.IsOK() || r.SafeVal("Name") != "inhere" {
					t.Errorf("Check(valid user): unexpected ok=%v name=%v", r.IsOK(), r.SafeVal("Name"))
				}
				if CheckErr(&checkAddr{City: "NYC", Zip: "10001"}) != nil {
					t.Errorf("CheckErr(valid addr): expected nil")
				}
				if r := Check(&checkAddr{City: "x"}); !r.Fail() {
					t.Errorf("Check(invalid addr): expected fail")
				}
			}
		}()
	}
	wg.Wait()
}

// =============================================================================
// scRV 缓存回归: struct 源同字段去重复用缓存的 reflect.Value(box-free, 免重读源),
// 并闭合非指针结构体的窄边界(scVal 重读源可能拿旧值, scRV 即已提交值更稳)。
// =============================================================================

// ceMultiNoDef: 无 default/无 filter 的同字段多规则, 走纯 dedup(无需写回源)。
// Age 的 min/max 两条规则: 第二条复用 scRV(第一条提交的字段 rv), box-free 免重读。
type ceMultiNoDef struct {
	Name string `validate:"required|min_len:3|max_len:10" json:"name"`
	Age  int    `validate:"required|min:1|max:150" json:"age"`
}

// TestCheckErr_scRV_structDedup: struct 源同字段多规则去重(scRV 路径)指针与非指针
// 入参都正确; 第二条规则用缓存 rv 校验, 值不丢失/不串味。
func TestCheckErr_scRV_structDedup(t *testing.T) {
	t.Run("ptr-valid", func(t *testing.T) {
		assert.NoErr(t, CheckErr(&ceMultiNoDef{Name: "inhere", Age: 30}))
	})
	t.Run("nonptr-valid", func(t *testing.T) {
		// 非指针入参: 无 default/filter 故无写回需求; Age 的 max:150 复用 scRV(30)。
		assert.NoErr(t, CheckErr(ceMultiNoDef{Name: "inhere", Age: 30}))
	})
	t.Run("nonptr-max-fail", func(t *testing.T) {
		// 第二条规则 max:150 用缓存 rv(999) 应失败 — 证明 scRV 携带的是真实字段值。
		assert.Err(t, CheckErr(ceMultiNoDef{Name: "inhere", Age: 999}))
	})
	t.Run("nonptr-min-fail", func(t *testing.T) {
		// Age=0 时 required 先失败; 用 1..150 区间外的 name 太短再触一条, 确认值正确。
		assert.Err(t, CheckErr(ceMultiNoDef{Name: "ab", Age: 30}))
	})
}

// TestCheckErr_scRV_nonPtr_vsCheck_differential: 非指针结构体入参下, CheckErr 与
// Check 结果逐项一致(differential)。覆盖 default/filter(非指针不可写回 -> 两路同机制
// 同样失败, 自洽)与纯多规则(两路同样通过)。scRV 让两路对"已提交值"的复用稳定一致。
func TestCheckErr_scRV_nonPtr_vsCheck_differential(t *testing.T) {
	cases := []func() any{
		// 纯多规则(无写回): 两路应一致通过/失败。
		func() any { return ceMultiNoDef{Name: "inhere", Age: 30} },
		func() any { return ceMultiNoDef{Name: "inhere", Age: 999} },
		func() any { return ceMultiNoDef{Name: "ab", Age: 30} },
		// default/filter 非指针不可写回: 两路同机制 -> 结果一致(此处均为失败)。
		func() any { return ceDefault{} },
		func() any { return ceFilterMulti{Name: " Inhere ", Tag: " A " } },
		func() any { return ceCrossField{Pwd: "abc123", Confirm: " abc123 "} },
	}
	for i, mk := range cases {
		ceErr := CheckErr(mk())
		ckRes := Check(mk())
		assert.Eq(t, ckRes.IsOK(), ceErr == nil,
			"nonptr case %d: CheckErr-ok=%v (err=%v) but Check.IsOK=%v", i, ceErr == nil, ceErr, ckRes.IsOK())
	}
}

// TestCheckErr_nonStruct: 非 struct 入参应返回非 nil error(FromStruct 的 ErrInvalidData
// 路径),不 panic。
func TestCheckErr_nonStruct(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		assert.Err(t, CheckErr(nil))
	})
	t.Run("map", func(t *testing.T) {
		assert.Err(t, CheckErr(map[string]any{"name": "inhere"}))
	})
	t.Run("int", func(t *testing.T) {
		assert.Err(t, CheckErr(123))
	})
	t.Run("string", func(t *testing.T) {
		assert.Err(t, CheckErr("not a struct"))
	})
}
