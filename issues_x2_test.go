package validate_test

import (
	"mime/multipart"
	"net/url"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2"
)

// https://github.com/gookit/validate/v2/issues/227
// 一个 key 包含多个上传文件时，除了第一个文件，其他文件被丢弃，导致 BindSafeData 行为非预期
func Test_Issue227(t *testing.T) {
	type UserForm struct {
		Name string
		File []*multipart.FileHeader
	}

	d := validate.FromURLValues(url.Values{
		"name": {"inhere"},
		"age":  {"30"},
	})
	// add files
	d.AddFile("File", &multipart.FileHeader{Filename: "test1.txt"}, &multipart.FileHeader{Filename: "test2.txt"})
	v := d.Create()
	v.AddRule("File", "min_len", 1)

	assert.True(t, v.Validate())
	dump.P(v.Errors)
	assert.Nil(t, v.Errors.ErrOrNil())

	u := &UserForm{}
	err := v.BindStruct(u)
	assert.NoError(t, err)
	dump.P(u)
}

// https://github.com/gookit/validate/v2/issues/259 Embedded structs are not validated properly #259
//
// not-a-bug in v2.0(核心问题已修): issue 标题"嵌入结构体未被正确验证"在 v1.5.2 下
// 的根因是匿名嵌入结构体的字段根本没有被级联验证。v2.0(T10)起匿名嵌入结构体始终级联
// (豁免 CheckSubOnParentMarked), 因此 NameField.Name / BranchField.Branch 上的
// `required` 规则现在会被求值 —— 嵌入字段已被正确验证。
//
// 报告者期望 perm2(Type=remove)整体通过, 其心智模型是: 嵌入字段 UserData 上的
// `required_if:Type,give` 不满足时, 应"门控"住其内部所有 required 规则。但 gookit
// 的设计是: 嵌入字段被展平后, 内部 `required` 规则是独立求值的, 不受外层
// required_if 门控。所以 perm2 仍会因 name/branch 为空而失败 —— 这是设计行为, 不是
// 缺陷; 正确写法应在内部字段上用条件规则。下面断言 v2.0 的真实行为。
func TestIssue_259_v2(t *testing.T) {
	type NameField struct {
		Name string `json:"name" validate:"required|max_len:5000"`
	}
	type BranchField struct {
		Branch string `json:"branch" validate:"required|min_len:32|max_len:32"`
	}
	type UserData struct {
		NameField   `json:",inline"`
		BranchField `json:",inline"`
	}
	type Permission struct {
		UserData `json:",inline" validate:"required_if:Type,give"`
		Type     string `json:"type" validate:"required|in:give,remove"`
		Access   string `json:"access" validate:"required_if:Type,remove"`
	}

	t.Run("embedded fields are now cascaded and validated (give+empty fails)", func(t *testing.T) {
		perm1 := Permission{UserData: UserData{}, Type: "give"}
		v1 := validate.Struct(perm1)
		v1.StopOnError = false
		// give 场景下 Name/Branch 必填, 空值应失败(证明嵌入字段确实被验证了)
		assert.False(t, v1.Validate())
		assert.ErrSubMsg(t, v1.Errors, "name is required")
	})

	t.Run("inner required is not gated by outer required_if (design behavior)", func(t *testing.T) {
		perm2 := Permission{Type: "remove", Access: "change_types"}
		v2 := validate.Struct(&perm2)
		v2.StopOnError = false
		// 嵌入字段被展平, 内部 required 独立求值, 不受外层 required_if 门控,
		// 因此 name/branch 为空仍报错 —— 这是设计行为, 非缺陷。
		assert.False(t, v2.Validate())
		assert.ErrSubMsg(t, v2.Errors, "name is required")
		assert.ErrSubMsg(t, v2.Errors, "branch is required")
	})
}

// https://github.com/gookit/validate/v2/issues/272 eqField对于指针类型数据无法正确校验
func Test_Issue272(t *testing.T) {
	type T272 struct {
		FieldA *string `validate:"required"`
		FieldB *string `validate:"required|eqField:FieldA"`
	}

	// test eqField
	var str = "abc"
	var str1 = "bcd"
	v := validate.Struct(&T272{
		FieldA: &str,
		FieldB: &str1,
	})
	assert.False(t, v.Validate())
	assert.Len(t, v.Errors, 1)
	assert.ErrSubMsg(t, v.Errors, "FieldB value must be equal the field FieldA")

	var str2 = "abc"
	v = validate.Struct(&T272{
		FieldA: &str,
		FieldB: &str2,
	})
	assert.True(t, v.Validate())
	assert.Nil(t, v.Errors.ErrOrNil())

	// nil value
	v = validate.Struct(&T272{
		FieldA: nil,
		FieldB: nil,
	})
	assert.False(t, v.Validate())
	assert.Len(t, v.Errors, 1)
	assert.ErrSubMsg(t, v.Errors, "FieldA is required")

}

// https://github.com/gookit/validate/v2/issues/316
// The int validator failed to validate a number exceeds the range of int64
func Test_Issue316(t *testing.T) {
	data := []byte(`{"value": 9223372036854775807}`)

	t.Run("not use filter", func(t *testing.T) {
		dataFace, err := validate.FromJSONBytes(data)
		assert.NoErr(t, err)

		v := dataFace.Create()
		v.StringRule("value", "int")
		assert.False(t, v.Validate())
		dump.P(v.Errors)
		assert.Err(t, v.Errors.ErrOrNil())
		assert.Equal(t, "value value must be an integer", v.Errors.One())
	})

	t.Run("use filter", func(t *testing.T) {
		dataFace, err := validate.FromJSONBytes(data)
		assert.NoErr(t, err)

		v := dataFace.Create()
		v.FilterRule("value", "int64")
		v.StringRule("value", "int")
		assert.True(t, v.Validate())
		assert.Nil(t, v.Errors.ErrOrNil())
		dump.P(v.SafeData())
	})
}

// https://github.com/gookit/validate/v2/issues/217 Nested resources are evaluated differently
//
// FIXED: Nested.Samples 是 []Sample, Sample.Val 是 *bool 且带 required。某元素的 Val
// 指向 false(非 nil 指针)时, 切片内子结构体的 required 曾把它误判为"空"而报错; 现已与
// 顶层路径一致, *bool->false 视为存在、正确通过。
//
// 根因: data_source.go 子结构体取值路径(tryGet 的 fieldAtSubStruct 分支)在导航循环里
// 把叶子 *bool 一并解引用成 bool(false), required 的 IsEmpty(bool(false))==true 误判为
// 空; 而顶层路径保留非 nil 指针(IsEmpty(非nil指针)==false), 两路径不一致 —— 这正是
// issue 标题 "Nested resources are evaluated differently" 所指。
//
// 修复: 子结构体导航循环对**叶子节点**保留非 nil 指针(不再解引用), 与顶层路径对齐;
// 中间节点仍解引用以便继续导航; nil 叶子指针仍按"不存在"处理。
func TestIssue_217_v2(t *testing.T) {
	type Sample struct {
		Val *bool `validate:"required"`
	}
	type Nested struct {
		Samples []Sample `validate:"slice"`
	}

	t.Run("top-level *bool->false passes required (correct)", func(t *testing.T) {
		val := false
		v := validate.Struct(Sample{Val: &val})
		// 顶层路径把 bool 视为存在, *bool->false 正确通过
		assert.True(t, v.Validate())
	})

	t.Run("nil *bool fails required (correct)", func(t *testing.T) {
		v := validate.Struct(Sample{Val: nil})
		assert.False(t, v.Validate())
	})

	t.Run("slice element *bool->false passes required (fixed)", func(t *testing.T) {
		val, val2 := false, true
		data := Nested{Samples: []Sample{{Val: &val}, {Val: &val2}}}
		v := validate.Struct(data)
		v.StopOnError = false
		// 两个元素的 Val 都非 nil, 应全部通过, 不再误报 Samples.0.Val 为空。
		ok := v.Validate()
		assert.True(t, ok)
		assert.Empty(t, v.Errors)
	})

	t.Run("slice element nil *bool still fails required (correct)", func(t *testing.T) {
		val := true
		data := Nested{Samples: []Sample{{Val: nil}, {Val: &val}}}
		v := validate.Struct(data)
		v.StopOnError = false
		ok := v.Validate()
		assert.False(t, ok)
		assert.ErrSubMsg(t, v.Errors, "Samples.0.Val is required")
		assert.NotContains(t, v.Errors.String(), "Samples.1.Val")
	})
}

// --- #235 support types (custom-validator method must be on a named pkg-level type) ---

type issue235Node struct {
	Name     string
	Location string
}

// issue235Config 用 issue 报告里推荐的"自定义校验器"写法验证 slice-of-struct。
// 注意 v2.0 下需要给 Nodes 字段加 validate tag(此处 customFunction)才会被收集。
type issue235Config struct {
	Nodes []issue235Node `validate:"customFunction"`
}

// CustomFunction 在父结构体上校验整个切片: 任一元素若有 Location 但缺 Name 则失败。
func (c issue235Config) CustomFunction(nodes []issue235Node) bool {
	for k := range nodes {
		if nodes[k].Location != "" && nodes[k].Name == "" {
			return false
		}
	}
	return true
}

func (c issue235Config) Messages() map[string]string {
	return validate.MS{"customFunction": "each {field} needs `Name` set if `Location` is set"}
}

// https://github.com/gookit/validate/v2/issues/235 How do I validate slice of struct inside a struct?
//
// partial in v2.0:
//   - 报告者最初的写法 `required_with:Nodes..Location`(跨切片元素的路径式 required_with)
//     仍不能按预期工作: 即便第 1 个元素没有 Location, 它仍报 Nodes.1.Name 必填。这条
//     路径式跨元素引用方式 v2.0 依旧不支持(still-broken, 见 t.Run "path-based...")。
//   - issue 自带的"在切片字段上挂自定义校验器"workaround 在 v2.0 下完全可用
//     (resolved): 给 Nodes 加 `validate:"customFunction"` tag 即可触发并逐元素校验,
//     这也是推荐解法。
func TestIssue_235_v2(t *testing.T) {
	t.Run("path-based required_with across slice elements still misbehaves", func(t *testing.T) {
		type Node struct {
			Name     string `validate:"required_with:Nodes..Location"`
			Location string
		}
		type Config struct {
			Nodes []Node `validate:""`
		}
		// 第 2 个元素既无 Location 也无 Name, 按 required_with 语义不该报错,
		// 但 v2.0 仍把 Nodes.1.Name 判为必填 —— 路径式跨元素引用未生效。
		data := &Config{Nodes: []Node{{Name: "node1", Location: "A"}, {}}}
		v := validate.Struct(data)
		v.StopOnError = false
		ok := v.Validate()
		t.Logf("v2.0 path-based required_with #235: ok=%v errors=%v", ok, v.Errors)
		assert.False(t, ok)
		assert.ErrSubMsg(t, v.Errors, "Nodes.1.Name")
	})

	t.Run("custom-validator workaround works (recommended)", func(t *testing.T) {
		// good: 每个有 Location 的元素都有 Name
		good := &issue235Config{Nodes: []issue235Node{{Name: "n1", Location: "A"}, {}}}
		vg := validate.Struct(good)
		vg.StopOnError = false
		assert.True(t, vg.Validate())

		// bad: 有元素含 Location 但缺 Name
		bad := &issue235Config{Nodes: []issue235Node{{Location: "B"}}}
		vb := validate.Struct(bad)
		vb.StopOnError = false
		assert.False(t, vb.Validate())
		assert.ErrSubMsg(t, vb.Errors, "needs `Name` set if `Location` is set")
	})
}

// https://github.com/gookit/validate/v2/issues/232
// 在校验 struct A 时, 其成员 a 是另一个 struct B 的 slice, 如何让 A.a 的 tag 触发对
// 每个 B 元素的字段规则(如 B 的 validateb)进行校验。
//
// resolved in v2.0: 只要给切片字段 a 加上 `validate` tag(空 tag `validate:""` 也算),
// v2.0 就会下探到每个 B 元素并应用 B 自身的字段规则(CheckSubOnParentMarked 行为)。
// 这正是该提问的答案: 给 A.a 加 validate tag 即可触发对 B 的级联验证; 不加 tag 则
// 不级联。
func TestIssue_232_v2(t *testing.T) {
	type B struct {
		Bval string `validate:"required|min_len:3"`
	}

	t.Run("slice field WITH validate tag cascades into each B", func(t *testing.T) {
		type A struct {
			Items []B `validate:"required"`
		}
		// 元素 Bval 太短, 应触发 B 的 min_len 规则
		v := validate.Struct(A{Items: []B{{Bval: "x"}}})
		v.StopOnError = false
		assert.False(t, v.Validate())
		assert.ErrSubMsg(t, v.Errors, "Items.0.Bval min length is 3")

		// 合法元素应通过
		ok := validate.Struct(A{Items: []B{{Bval: "hello"}}})
		assert.True(t, ok.Validate())
	})

	t.Run("slice field WITHOUT validate tag does not cascade (v2.0 design)", func(t *testing.T) {
		type A struct {
			Items []B
		}
		// 无 validate tag, v2.0 不下探, 即便元素非法也通过
		v := validate.Struct(A{Items: []B{{Bval: "x"}}})
		v.StopOnError = false
		assert.True(t, v.Validate())
	})
}

// --- #283 support type (ConfigValidation/CustomValidator methods need pkg-level type) ---

type issue283Tag struct {
	Id   string `validate:"required"`
	Name string `validate:"required"`
	Date string `validate:"required"`
}

type issue283Form struct {
	Name string        `validate:"required|min_len:7"`
	Code string        `validate:"required|customValidator"`
	Tags []issue283Tag `validate:"required"`
	Test int           `validate:"required|greaterThan:1"`
}

func (f issue283Form) CustomValidator(val string) bool { return len(val) == 4 }

func (f issue283Form) ConfigValidation(v *validate.Validation) {
	v.WithScenes(validate.SValues{
		// 报告者写法: 用不带索引的 "Tags.Id" 想校验切片元素
		"update":     []string{"Name", "Tags.Id", "Test"},
		"updateStar": []string{"Tags.*.Id"},
		"updateIdx":  []string{"Tags.0.Id"},
	})
}

// https://github.com/gookit/validate/v2/issues/283 Scenes does not work in slices
//
// FIXED(通配 .*): 级联为切片元素生成带索引的规则名 "Tags.0.Id", 场景现支持用通配
// "Tags.*.Id" 命中它 —— 把字段名的数字索引段规范化为 "*" 后与场景通配项比对。这正是
// issue 标题 "Scenes does not work in slices" 的核心诉求(选择性校验切片元素的某个字段)。
//
// 范围: 仅支持 ".*" 通配写法。**无索引** "Tags.Id" 仍不命中(与真实嵌套字段 "Tags.Id"
// 有语义歧义, 按设计不做), 推荐统一用 "Tags.*.Id"。显式索引 "Tags.0.Id" 仍可用。
func TestIssue_283_v2(t *testing.T) {
	newForm := func() *issue283Form {
		return &issue283Form{
			Name: "inhere", Code: "asd", Test: 1,
			Tags: []issue283Tag{{Id: "", Name: "", Date: ""}},
		}
	}

	t.Run("scene 'Tags.Id' index-less still does NOT match (by design, use .*)", func(t *testing.T) {
		v := validate.Struct(newForm(), "update")
		v.StopOnError = false
		ok := v.Validate("update")
		// Name(太短)与 Test 报错, 但 Tags.0.Id 不被无索引 "Tags.Id" 命中(无 Tags 错误)
		assert.False(t, ok)
		assert.NotContains(t, v.Errors.String(), "Tags")
	})

	t.Run("scene 'Tags.*.Id' wildcard validates slice element (fixed)", func(t *testing.T) {
		v := validate.Struct(newForm(), "updateStar")
		v.StopOnError = false
		ok := v.Validate("updateStar")
		// 通配命中 Tags.0.Id, 空 Id 正确报错; Name/Date 不在场景内, 不被校验
		assert.False(t, ok)
		assert.ErrSubMsg(t, v.Errors, "Tags.0.Id is required")
		assert.NotContains(t, v.Errors.String(), "Tags.0.Name")
		assert.NotContains(t, v.Errors.String(), "Tags.0.Date")
		// Name 太短(min_len:7)但不在 updateStar 场景内, 不应报错
		assert.NotContains(t, v.Errors.String(), "Name")
	})

	t.Run("scene 'Tags.0.Id' explicit index DOES validate (workaround still valid)", func(t *testing.T) {
		v := validate.Struct(newForm(), "updateIdx")
		v.StopOnError = false
		ok := v.Validate("updateIdx")
		// 显式数字索引可命中, 空 Id 正确报错
		assert.False(t, ok)
		assert.ErrSubMsg(t, v.Errors, "Tags.0.Id is required")
	})
}

// --- #314 support types (ConfigValidation needs pkg-level type) ---

type issue314Sub struct {
	A string
}

type issue314Struct struct {
	SubStruct *issue314Sub `validate:"required"`
}

func (issue314Struct) ConfigValidation(v *validate.Validation) {
	v.WithScenes(validate.SValues{
		"SubStruct": []string{"SubStruct"},
		"None":      []string{""}, // 期望: 此场景下什么都不校验
	})
}

// https://github.com/gookit/validate/v2/issues/314 Scene with empty validation rules now fails
//
// FIXED: 场景 "None": []string{""} 表示"不校验任何字段", 现已恢复 v1.4.5 行为, 不再误报
// "SubStruct is required"。
//
// 根因: sceneFieldMap() 曾把空字符串 "" 当作普通场景字段键, 得到 sceneFields={"":1}; 随后
// isNotNeedToCheck("SubStruct") 里 `strings.Join(fields[0:0], ".")` 得到 "" 恰好命中,
// 于是所有字段反而都被纳入校验。
//
// 修复: sceneFieldMap() 跳过空字段(但保留非 nil map 以区分"无场景"); isNotNeedToCheck()
// 用 nil/非nil 区分"未设场景(校验全部)"与"场景已激活但无字段(什么都不校验)"。
func TestIssue_314_v2(t *testing.T) {
	t.Run("empty-rule scene 'None' validates nothing (fixed)", func(t *testing.T) {
		foo := issue314Struct{}
		err := validate.Struct(&foo).ValidateE("None")
		// 期望: err 为空(此场景什么都不校验)。
		assert.Empty(t, err)
	})

	t.Run("scene 'SubStruct' validates SubStruct (correct)", func(t *testing.T) {
		foo := issue314Struct{}
		err := validate.Struct(&foo).ValidateE("SubStruct")
		assert.NotEmpty(t, err)
		assert.StrContains(t, err.String(), "SubStruct is required")
	})
}

// https://github.com/gookit/validate/v2/issues/327 未指定 required 时没有获取到对应字段的值
//
// not-a-bug in v2.0(设计行为, 已可正常工作): 报告者抱怨"未填写 required 的字段
// (如 bbb)取不到绑定值"。实际规则是: 只有"参与了校验且通过"的字段才会进入 safeData,
// 进而能被 BindStruct 绑定。
//   - bbb 加了非 required 规则(如 string)且有非空值 -> 正常进入 safeData 并绑定(OK)。
//   - bbb 完全没有规则, 或值为空被 SkipOnEmpty 跳过 -> 不进入 safeData, 自然取不到。
//
// 这与 required 无关, 是"safeData 只含校验过的数据"的设计。报告者把它当 BUG 实为
// 用法误解。AddMessages 不写 required 项也不会影响其它字段的消息(下面一并验证)。
func TestIssue_327_v2(t *testing.T) {
	t.Run("non-required rule with value IS captured & bound", func(t *testing.T) {
		data := validate.FromMap(map[string]any{"aaa": "hello", "bbb": "world"})
		v := data.Create()
		v.StringRule("aaa", "required|string|minLen:4")
		v.StringRule("bbb", "string") // 非 required 规则
		assert.True(t, v.Validate())

		var reqData struct {
			Aaa string `json:"aaa"`
			Bbb string `json:"bbb"`
		}
		assert.NoErr(t, v.BindStruct(&reqData))
		// 非 required 的 bbb 同样被绑定到了
		assert.Eq(t, "hello", reqData.Aaa)
		assert.Eq(t, "world", reqData.Bbb)
		assert.Eq(t, "world", v.SafeData()["bbb"])
	})

	t.Run("field WITHOUT any rule is not in safeData (design)", func(t *testing.T) {
		data := validate.FromMap(map[string]any{"aaa": "hello", "bbb": "world"})
		v := data.Create()
		v.StringRule("aaa", "required|string|minLen:4")
		// bbb 没有任何规则
		assert.True(t, v.Validate())
		// 没有规则的字段不进入 safeData -> 取不到, 这是设计而非缺陷
		_, ok := v.SafeData()["bbb"]
		assert.False(t, ok)
	})

	t.Run("AddMessages without 'required' entry does not break other field messages", func(t *testing.T) {
		data := validate.FromMap(map[string]any{"aaa": "x", "bbb": "world"})
		v := data.Create()
		v.StringRule("aaa", "required|string|minLen:4")
		v.StringRule("bbb", "string")
		// 故意不提供 aaa.required 的自定义消息
		v.AddMessages(map[string]string{
			"aaa.minLen": "长度不少于 4 个字符",
			"bbb.string": "格式不正确",
		})
		assert.False(t, v.Validate())
		// aaa.minLen 的自定义消息仍生效, 其它字段消息未被吞掉
		assert.StrContains(t, v.Errors.String(), "长度不少于 4 个字符")
	})
}

// https://github.com/gookit/validate/v2/issues/262 filter 无法对切片中的元素应用
//
// FIXED: 通配路径 "ports.*.container_start" 给切片元素挂 filter, 现已逐元素应用,
// 不再报 `convert value type error`。
//
// 根因: 通配路径经 GetByPath 取到整个切片 []any{80}(而非逐个标量), 旧逻辑把 filter("int")
// 应用到整个 slice -> 当单个值转 int 失败。filter 链路缺少验证器侧 validateWildcardSlice
// 那样的 ".*" 逐元素展开。
//
// 修复: FilterRule.Apply 检测到字段含 ".*" 且取值为切片时, 逐元素 apply filter, 结果作为
// 收集切片写回 filteredData[通配key]; 验证器经 tryGet 优先读 filteredData, 形态天然对齐,
// 逐元素校验直接跑通。范围: 先支持 Map/JSON 源、单级 ".*"。
func TestIssue_262_v2(t *testing.T) {
	jsonStr := `{"ports":[{"container_start":80,"container_end":81,"protocol":"tcp"},{"container_start":90,"container_end":91,"protocol":"udp"}]}`

	t.Run("wildcard filter on slice elements applies per-element (fixed)", func(t *testing.T) {
		v, err := validate.FromJSON(jsonStr)
		assert.NoErr(t, err)
		vv := v.Create()
		vv.FilterRule("ports.*.container_start", "int")
		vv.FilterRule("ports.*.container_end", "int")
		vv.AddRule("ports.*.container_start", "int")
		vv.AddRule("ports.*.container_end", "int")
		vv.AddRule("ports.*.protocol", "string")

		ok := vv.Validate()
		assert.True(t, ok)
		assert.Empty(t, vv.Errors)
		// 逐元素转为 int 后, 收集切片写回到通配 key
		starts, hasStart := vv.SafeData()["ports.*.container_start"]
		assert.True(t, hasStart)
		assert.Eq(t, []any{80, 90}, starts)
	})

	t.Run("explicit index filter works (workaround still valid)", func(t *testing.T) {
		v, err := validate.FromJSON(jsonStr)
		assert.NoErr(t, err)
		vv := v.Create()
		// 显式数字索引可命中标量值, filter 正常
		vv.FilterRule("ports.0.container_start", "int")
		vv.AddRule("ports.0.container_start", "int")
		assert.True(t, vv.Validate())
		assert.Eq(t, 80, vv.SafeData()["ports.0.container_start"])
	})

	t.Run("wildcard custom filter applies per-element (fixed)", func(t *testing.T) {
		v, err := validate.FromJSON(jsonStr)
		assert.NoErr(t, err)
		vv := v.Create()
		// 自定义 filter 走 callCustomFilter 分支, 同样应逐元素生效
		vv.AddFilter("plus10", func(val any) (int, error) {
			n, e := mathutil.ToInt(val)
			return n + 10, e
		})
		vv.FilterRule("ports.*.container_start", "plus10")
		vv.AddRule("ports.*.container_start", "int")
		assert.True(t, vv.Validate())
		assert.Eq(t, []any{90, 100}, vv.SafeData()["ports.*.container_start"])
	})
}

// https://github.com/gookit/validate/v2/issues/138 FullUrl regex needs improvement
//
// still-broken in v2.0(enhancement 未做): fullUrl 的正则
//
//	^(?:ftp|tcp|udp|wss?|https?):\/\/[\w\.\/#=?&-_%]+$
//
// 字符类过于宽松, 把一批非法 URL 判为合法。issue 给的三个反例当前都通过 IsFullURL:
//   - "https://www.googl_?e.com/testme"(含 _ 与 ? 等)
//   - "https://www"(无 TLD)
//   - "https://not%23"
//
// 此外 IsURL 基于 url.Parse(err==nil), 更宽松(连 "not a url" 都判为合法)。
//
// 修复方向(改业务代码, 本任务只核查): 参考 asaskevich/govalidator 的 URL 正则收紧
// host/TLD/字符集校验。故此处断言 v2.0 真实(仍宽松)行为并标注。
func TestIssue_138_v2(t *testing.T) {
	t.Run("fullUrl regex too permissive, accepts invalid URLs (still-broken)", func(t *testing.T) {
		invalidButAccepted := []string{
			"https://www.googl_?e.com/testme",
			"https://www",
			"https://not%23",
		}
		for _, s := range invalidButAccepted {
			// 期望: 应为 false(非法)。实际: 仍 true。断言现状。
			assert.True(t, validate.IsFullURL(s), "expected v2.0 to (wrongly) accept %q", s)
		}

		// 合法基线仍应通过
		for _, s := range []string{"http://example.com", "https://www.google.com/testme", "ftp://files.example.com/a"} {
			assert.True(t, validate.IsFullURL(s))
		}
	})

	t.Run("IsURL (url.Parse based) even more permissive (still-broken)", func(t *testing.T) {
		// url.Parse 几乎不报错, 连明显非 URL 的串也判为合法
		assert.True(t, validate.IsURL("not a url"))
	})
}

// --- #324 support types (nested form-data binding) ---

type issue324Address struct {
	Street string `form:"street" json:"street"`
	City   string `form:"city" json:"city"`
}

type issue324Member struct {
	Name    string          `form:"name" json:"name"`
	Address issue324Address `form:"address" json:"address"`
}

// https://github.com/gookit/validate/v2/issues/324 Nested Form Data Binding Fails for multipart/form-data
//
// still-broken in v2.0(未支持的特性): multipart/form-data 的嵌套字段(无论 bracket 写法
// "address[street]" 还是 dot 写法 "address.street")都不能绑定到嵌套 struct。
//
// 根因: FromURLValues 把表单键**原样平铺**进 d.Form(url.Values); BindSafeData 把
// safeData(平铺 map)json.Marshal 再 Unmarshal —— 平铺键 "address[street]"/"address.street"
// 在 JSON 里只是顶层扁平键, 不会变成嵌套对象, 故 Address 子字段始终为空。validate 没有
// 对 bracket/dot 嵌套键做解析或展开。
//
// 修复方向(改业务代码, 本任务只核查): 绑定前把 bracket "a[b]" 归一为路径 "a.b", 并按点
// 路径把平铺键展开成嵌套 map 再绑定。此处断言 v2.0 真实(嵌套不绑定)行为并标注。
func TestIssue_324_v2(t *testing.T) {
	run := func(t *testing.T, street, city string) issue324Member {
		t.Helper()
		form := url.Values{}
		form.Set("name", "John")
		form.Set(street, "Main St")
		form.Set(city, "New York")

		data := validate.FromURLValues(form)
		v := data.Create()
		v.StringRule("name", "required")
		assert.True(t, v.Validate())

		var req issue324Member
		assert.NoErr(t, v.BindSafeData(&req))
		return req
	}

	t.Run("bracket notation address[street] does NOT bind nested (still-broken)", func(t *testing.T) {
		req := run(t, "address[street]", "address[city]")
		// 简单字段正常绑定
		assert.Eq(t, "John", req.Name)
		// 嵌套字段未绑定(期望 Main St / New York)
		assert.Eq(t, "", req.Address.Street)
		assert.Eq(t, "", req.Address.City)
	})

	t.Run("dot notation address.street also does NOT bind nested (still-broken)", func(t *testing.T) {
		req := run(t, "address.street", "address.city")
		assert.Eq(t, "John", req.Name)
		assert.Eq(t, "", req.Address.Street)
		assert.Eq(t, "", req.Address.City)
	})
}

// https://github.com/gookit/validate/v2/issues/277 Identifying the First Failed Field (StopOnError=true)
//
// answered(非 bug): StopOnError=true(默认)时 Validate 失败后, v.Errors 仅含**第一个**出错
// 字段; 可直接取字段名 + 校验器(tag) + 消息。这就是提问的答案。
func TestIssue_277_v2(t *testing.T) {
	type SomeStruct struct {
		Name  string `validate:"required"`
		Email string `validate:"required|email"`
	}
	// Name 合法(required 通过), Email 非空但非邮箱 -> email 校验失败
	v := validate.Struct(&SomeStruct{Name: "John", Email: "not an email"})
	assert.True(t, v.StopOnError) // 默认开启
	assert.False(t, v.Validate())

	// 仅第一个出错字段 Email 进入 Errors, Name 不在
	assert.True(t, v.Errors.HasField("Email"))
	assert.False(t, v.Errors.HasField("Name"))
	// 取该字段失败的校验器(tag)与消息
	fe := v.Errors.Field("Email") // map[validator]message
	_, byEmail := fe["email"]
	assert.True(t, byEmail)
	assert.StrContains(t, v.Errors.FieldOne("Email"), "Email")
}

// https://github.com/gookit/validate/v2/issues/266 Can `in` be used with slice in tag validation?
//
// partial(答案=用 .* 程序化写法): 结构体 tag 里的 in(enum)不会逐元素校验切片, 而是把整个
// []string 当单值丢给 Enum -> ConvToBasicType 失败 -> 整体报错。要逐元素校验需用程序化的
// "S.*" 规则(StringRule), tag 方式不支持。
func TestIssue_266_v2(t *testing.T) {
	t.Run("tag 'in' on []string checks whole slice and fails (not per-element)", func(t *testing.T) {
		type A struct {
			S []string `validate:"required|in:a,b"`
		}
		// 两个元素都合法, 但 in 作用于整个切片 -> 仍失败
		v := validate.Struct(&A{S: []string{"a", "b"}})
		v.StopOnError = false
		assert.False(t, v.Validate())
		assert.ErrSubMsg(t, v.Errors, "must be in the enum")
	})

	t.Run("workaround: programmatic 'S.*' validates each element", func(t *testing.T) {
		ok := validate.Map(map[string]any{"S": []string{"a", "b"}})
		ok.StringRule("S.*", "in:a,b")
		assert.True(t, ok.Validate())

		bad := validate.Map(map[string]any{"S": []string{"a", "x"}})
		bad.StringRule("S.*", "in:a,b")
		assert.False(t, bad.Validate())
		assert.ErrSubMsg(t, bad.Errors, "must be in the enum")
	})
}

// https://github.com/gookit/validate/v2/issues/265 How CheckZero config flag works.
//
// finding -> Deprecated: GlobalOption.CheckZero 已声明但无任何消费方(no-op), 切换它不会
// 改变任何校验行为。已在 v2.0 标记为 Deprecated(见 validate.go), 后续移除。零值是否参与
// 校验请用 Rule.SetSkipEmpty(false) / SkipOnEmpty=false / required, 与 CheckZero 无关。
//
// 本用例**有意**设置该 deprecated 字段, 以证明其为 no-op(切换前后结果一致)。
func TestIssue_265_v2(t *testing.T) {
	type Foo struct {
		Age int `validate:"min:18"`
	}
	run := func() bool { return validate.Struct(&Foo{Age: 0}).Validate() }

	got1 := run() // CheckZero=false(默认)
	validate.Config(func(opt *validate.GlobalOption) { opt.CheckZero = true }) //nolint:staticcheck // intentionally exercise deprecated no-op
	defer validate.ResetOption()
	got2 := run() // CheckZero=true

	// 切换 CheckZero 前后结果完全一致 -> 该 flag 当前无效果
	assert.Eq(t, got1, got2)
}

// https://github.com/gookit/validate/v2/issues/162 Combine required_if with a validation rule
//
// answered(可实现): required_if 可与后续规则链式组合。"required_if:Type,B|uuid4" 表示
// 当 Type==B 时 ID 必填且须为 uuid4; Type!=B 时 ID 可空(按 SkipOnEmpty 跳过), 但若提供
// 非空值仍按 uuid4 校验(uuid4 不被条件门控, 仅"是否必填"被门控)。
func TestIssue_162_v2(t *testing.T) {
	type Form struct {
		Type string `validate:"in:B,C"`
		ID   string `validate:"required_if:Type,B|uuid4"`
	}
	good := "94e48bd3-e990-405e-bd10-304e767cd3fd"

	t.Run("Type=B + valid uuid -> pass", func(t *testing.T) {
		assert.True(t, validate.Struct(&Form{Type: "B", ID: good}).Validate())
	})
	t.Run("Type=B + empty -> fail (required)", func(t *testing.T) {
		v := validate.Struct(&Form{Type: "B", ID: ""})
		v.StopOnError = false
		assert.False(t, v.Validate())
		assert.ErrSubMsg(t, v.Errors, "ID is required")
	})
	t.Run("Type=C + empty -> pass (not required)", func(t *testing.T) {
		assert.True(t, validate.Struct(&Form{Type: "C", ID: ""}).Validate())
	})
	t.Run("Type=B + invalid uuid -> fail (uuid4)", func(t *testing.T) {
		v := validate.Struct(&Form{Type: "B", ID: "notauuid"})
		v.StopOnError = false
		assert.False(t, v.Validate())
		assert.ErrSubMsg(t, v.Errors, "UUID4")
	})
	t.Run("Type=C + non-empty invalid uuid -> still fail uuid4 (rule not gated by condition)", func(t *testing.T) {
		v := validate.Struct(&Form{Type: "C", ID: "notauuid"})
		v.StopOnError = false
		assert.False(t, v.Validate())
		assert.ErrSubMsg(t, v.Errors, "UUID4")
	})
}
