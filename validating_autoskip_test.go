package validate

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// 透明优化测试:struct 源在只返回 bool/error 的入口(Validate/ValidateErr/ValidateE)
// 自动走 skipCollect 快路径(免 safeData 收集装箱);ValidateR/Check 经 needCollect
// 强制收集;map/form 源 及 UpdateSource=false 不跳过。

type asUser struct {
	Name  string `validate:"required|min_len:3" json:"name"`
	Email string `validate:"required|email" json:"email"`
	Age   int    `validate:"required|min:1|max:120" json:"age"`
}

func validAsUser() *asUser {
	return &asUser{Name: "inhere", Email: "test@example.com", Age: 30}
}

// struct + filter(trim/int) + 跨字段(eq_field) 的差分用例。
type asCross struct {
	Pwd     string `validate:"required" json:"pwd"`
	Confirm string `validate:"required|eq_field:Pwd" filter:"trim" json:"confirm"`
}

// struct + filter(int) + 跨字段(gt_field)。
type asGt struct {
	Min int `validate:"required|int" json:"min"`
	Max int `validate:"required|int|gt_field:Min" json:"max"`
}

// 1) struct ValidateErr/Validate 合法 → 通过;且分配数 < 收集模式(ValidateR)。
func TestAutoSkip_structValidateErrPasses(t *testing.T) {
	t.Run("ValidateErr ok", func(t *testing.T) {
		assert.NoErr(t, Struct(validAsUser()).ValidateErr())
	})
	t.Run("Validate ok", func(t *testing.T) {
		assert.True(t, Struct(validAsUser()).Validate())
	})
	t.Run("ValidateE ok", func(t *testing.T) {
		assert.Empty(t, Struct(validAsUser()).ValidateE())
	})

	t.Run("allocs skip < collect", func(t *testing.T) {
		// 自动跳过(struct 源 ValidateErr):不收集 safeData。
		skipAllocs := testing.AllocsPerRun(200, func() {
			_ = Struct(validAsUser()).ValidateErr()
		})
		// 强制收集(ValidateR 经 needCollect):收集 safeData。
		collectAllocs := testing.AllocsPerRun(200, func() {
			_ = Struct(validAsUser()).ValidateR()
		})
		t.Logf("skipAllocs=%.0f collectAllocs=%.0f", skipAllocs, collectAllocs)
		// 不苛求 0(非池化 *Validation 自身有开销),只断言跳过模式分配更少,
		// 证明 safeData 装箱收集被省去。
		assert.Lt(t, skipAllocs, collectAllocs)
	})
}

// 2) struct ValidateR/Check → SafeData 仍正确非空(收集生效,未被误跳)。
func TestAutoSkip_structValidateRStillCollects(t *testing.T) {
	t.Run("ValidateR", func(t *testing.T) {
		r := Struct(validAsUser()).ValidateR()
		assert.True(t, r.IsOK())
		assert.NotEmpty(t, r.SafeData())
		assert.Eq(t, "inhere", r.SafeVal("Name"))
		assert.Eq(t, 30, r.SafeVal("Age"))
	})
	t.Run("Check", func(t *testing.T) {
		r := Check(validAsUser())
		assert.True(t, r.IsOK())
		assert.NotEmpty(t, r.SafeData())
		assert.Eq(t, "inhere", r.SafeVal("Name"))
	})
}

//  3. struct + filter + 跨字段:差分——同数据分别走 ValidateErr(跳过)与
//     ValidateR(收集),断言过/败一致(写回保证跨字段正确)。
func TestAutoSkip_structFilterCrossFieldDiff(t *testing.T) {
	cases := []struct {
		name string
		gen  func() any // 每次新建(写回会改写源,见 result_test 说明)
	}{
		{"cross eq pass (trim)", func() any { return &asCross{Pwd: "secret", Confirm: " secret "} }},
		{"cross eq fail", func() any { return &asCross{Pwd: "secret", Confirm: "other"} }},
		{"gt pass", func() any { return &asGt{Min: 1, Max: 5} }},
		{"gt fail", func() any { return &asGt{Min: 5, Max: 1} }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			skipErr := Struct(c.gen()).ValidateErr() // 自动跳过
			collectOK := Struct(c.gen()).ValidateR().IsOK()
			// 过/败一致:跳过路径(skipErr==nil)与收集路径(collectOK)结论相同。
			assert.Eq(t, collectOK, skipErr == nil)
		})
	}

	t.Run("trimmed value written back & exposed in collect", func(t *testing.T) {
		r := Struct(&asCross{Pwd: "secret", Confirm: " secret "}).ValidateR()
		assert.True(t, r.IsOK())
		assert.Eq(t, "secret", r.SafeVal("Confirm")) // trim 后写回+收集
	})
}

// 4) map 源 ValidateErr → 不跳过,行为不变(含 filter + 跨字段)。
// 白盒:同包内直接读 v.skipCollect / v.safeData 证明未被跳过、仍收集。
func TestAutoSkip_mapSourceNotSkipped(t *testing.T) {
	t.Run("map cross-field pass + collect", func(t *testing.T) {
		v := Map(map[string]any{"pwd": "secret", "confirm": "secret"})
		v.StringRule("pwd", "required")
		v.StringRule("confirm", "required|eq_field:pwd")
		assert.NoErr(t, v.ValidateErr())
		assert.False(t, v.skipCollect) // map 源:gate 不触发
		assert.NotEmpty(t, v.safeData) // 仍收集(行为不变)
	})
	t.Run("map cross-field fail", func(t *testing.T) {
		v := Map(map[string]any{"pwd": "secret", "confirm": "other"})
		v.StringRule("pwd", "required")
		v.StringRule("confirm", "required|eq_field:pwd")
		assert.Err(t, v.ValidateErr())
	})
	t.Run("map filter still applied + collect", func(t *testing.T) {
		v := Map(map[string]any{"name": "  Tom  "})
		v.StringRule("name", "required|min_len:3", "trim")
		assert.NoErr(t, v.ValidateErr())
		assert.False(t, v.skipCollect)
		assert.Eq(t, "Tom", v.safeData["name"]) // 收集生效(trim 后)
	})
}

// 5) struct 但 UpdateSource=false → 不跳过(collect),结果正确。
func TestAutoSkip_updateSourceFalseNotSkipped(t *testing.T) {
	v := Struct(validAsUser())
	v.UpdateSource = false // 关闭写回 → gate 不触发,正常收集
	assert.True(t, v.Validate())
	assert.False(t, v.skipCollect) // gate 未触发
	assert.NotEmpty(t, v.safeData) // 仍收集
	assert.Eq(t, "inhere", v.safeData["Name"])
}

// 6) struct 只读入口确实触发了 skipCollect 快路径(白盒确证)。
func TestAutoSkip_structReadonlyTriggersSkip(t *testing.T) {
	v := Struct(validAsUser())
	assert.NoErr(t, v.ValidateErr())
	assert.True(t, v.skipCollect) // 自动跳过已触发
	assert.Empty(t, v.safeData)   // 未收集 safeData
}
