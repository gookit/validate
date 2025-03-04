package validate

import (
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/testutil/assert"
)

func TestNew(t *testing.T) {
	v := New(map[string][]string{
		"age":  {"12"},
		"name": {"inhere"},
	})
	v.StringRules(MS{
		"age":  "required|strInt",
		"name": "required|string:3|strLen:4,6",
	})

	assert.True(t, v.Validate())
}

func TestRule_Apply(t *testing.T) {
	is := assert.New(t)
	mp := M{
		"name": "inhere",
		"code": "2363",
	}

	v := Map(mp)
	v.ConfigRules(MS{
		"name": `regex:\w+`,
	})
	v.AddRule("name", "stringLength", 3)
	v.StringRule("code", `required|regex:\d{4,6}`)

	is.True(v.Validate())
}

func TestStruct_useDefault(t *testing.T) {
	is := assert.New(t)

	type user struct {
		Name string `validate:"required|default:tom" filter:"trim|upper"`
		Age  int    `validate:"uint|default:23"`
	}

	u := &user{Age: 90}
	v := New(u)
	is.True(v.Validate())
	is.Equal("tom", u.Name)

	u = &user{Name: "inhere"}
	v = New(u)
	is.True(v.Validate())
	// fmt.Println(v.SafeData())
}

func TestValidation_CheckDefault(t *testing.T) {
	is := assert.New(t)

	type user struct {
		Name string `validate:"required|default:tom" filter:"trim|upper"`
		Age  int    `validate:"uint|default:23"`
	}

	// check/filter default value
	u := &user{Age: 90}
	v := New(u).WithSelf(func(v *Validation) {
		v.CheckDefault = true
	})

	is.True(v.Validate())
	is.Equal("TOM", u.Name)
}

func TestValidation_RequiredIf(t *testing.T) {
	// test map data
	v := New(M{
		"name": "lee",
		"age":  "12",
	})
	v.StringRules(MS{
		"age":     "required_if:name,lee",
		"nothing": "required_if:age,12,13,14",
	})

	assert.False(t, v.RequiredIf("age", "12"))
	assert.False(t, v.Validate())
	assert.Equal(t, "nothing is required when age is in [12,13,14]", v.Errors.One())

	// test struct data
	type user struct {
		Name string `validate:"required"`
		Age  int    `validate:"required_if:Name,lee"`
	}

	u := &user{Name: "lee"}
	v = New(u)
	e := v.ValidateErr()
	assert.Err(t, e)
	assert.StrContains(t, e.Error(), "Age is required when Name is in [lee]")
}

// 验证的字段必须存在且不为空，除非目标字段等于给定的任何值。
func TestValidation_RequiredUnless(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "lee",
		"nothing": "",
	})
	v.StringRules(MS{
		"age":     "required_unless:name,lee",
		"nothing": "required_unless:age,12,13,14",
	})

	assert.False(t, v.RequiredUnless("age", "12"))
	assert.False(t, v.Validate())
	assert.Equal(t, "nothing field is required unless age is in [12,13,14]", v.Errors.One())

	// test struct data
	type user struct {
		Name string `validate:"required"`
		Age  int    `validate:"required_unless:Name,tom"`
	}

	u := &user{Name: "lee"}
	v = New(u)
	assert.False(t, v.Validate())
	assert.Equal(t, "Age field is required unless Name is in [tom]", v.Errors.One())
}

// 仅当任何其他指定字段存在时，验证下的字段才必须存在且不为空。
func TestValidation_RequiredWith(t *testing.T) {
	// test map data
	v := New(M{
		"age":  "18",
		"name": "test",
	})
	v.StringRules(MS{
		"age":      "required_with:name,city",
		"anything": "required_with:sex,city",
		"nothing":  "required_with:age,name",
	})

	assert.False(t, v.RequiredWith("age", "12"))
	assert.False(t, v.Validate())
	assert.Equal(t, "nothing field is required when [age,name] is present", v.Errors.One())

	// test struct data
	type user struct {
		Name string `validate:"requiredWith:Age,City"`
		Age  int    `validate:"required_with:Name,City"`
		City string `validate:"required_with:Name,Age"`
	}

	u := &user{Name: "lee"}
	v = New(u)
	assert.False(t, v.Validate())
	assert.Equal(t, "Age field is required when [Name,City] is present", v.Errors.One())
}

// 仅当所有其他指定字段都存在时，验证下的字段才必须存在且不为空。
func TestValidation_RequiredWithAll(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "test",
		"sex":     "man",
		"nothing": "",
	})
	v.StringRules(MS{
		"age":      "required_with_all:name,sex,city",
		"anything": "required_with_all:school,city",
		"nothing":  "required_with_all:age,name,sex",
	})

	assert.False(t, v.RequiredWithAll("age", "12"))
	assert.False(t, v.Validate())
	assert.Equal(t, "nothing field is required when [age,name,sex] is present", v.Errors.One())

	// test struct data
	type user struct {
		Age  int
		Sex  string
		Name string `validate:"requiredWithAll:age,sex"`
	}

	u := &user{Age: 23}
	v = New(u)
	assert.True(t, v.Validate())
	u.Sex = "man"
	v = New(u)
	assert.False(t, v.Validate())
	assert.Equal(t, "Name field is required when [age,sex] is present", v.Errors.One())
}

// 仅当任何一个其他指定字段不存在时，验证中的字段才必须存在且不为空。
func TestValidation_RequiredWithout(t *testing.T) {
	v := New(M{
		"age":  "18",
		"name": "test",
	})
	v.StringRules(MS{
		"age":      "required_without:city",
		"anything": "required_without:age,name",
		"nothing":  "required_without:sex,name",
	})

	assert.False(t, v.RequiredWithout("age", "12"))
	assert.False(t, v.Validate())
	assert.Equal(t, "nothing field is required when [sex,name] is not present", v.Errors.One())

	// test struct data
	type user struct {
		Name string `validate:"requiredWithout:Age,City"`
		Age  int    `validate:"required_without:Name,City"`
		City string
	}

	u := &user{Name: "lee"}
	v = New(u)
	assert.False(t, v.Validate())
	assert.Equal(t, "Age field is required when [Name,City] is not present", v.Errors.One())
}

// 仅当任何其他指定字段都不存在时，验证中的字段才必须存在且不为空。
func TestValidation_RequiredWithoutAll(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "test",
		"nothing": "",
	}).StringRules(MS{
		"age":      "required_without_all:name,city",
		"anything": "required_without:age,name",
		"nothing":  "required_without_all:sex,city",
	})

	assert.False(t, v.RequiredWithoutAll("age", "12"))
	assert.False(t, v.Validate())
	assert.Equal(t, "nothing field is required when none of [sex,city] are present", v.Errors.One())

	// test struct data
	type user struct {
		Name string `validate:"requiredWithoutAll:Age,City"`
		Age  int
		City string
	}

	u := &user{Name: "lee"}
	v = New(u)
	assert.True(t, v.Validate())
	u.Name = ""
	v = New(u)
	assert.False(t, v.Validate())
	assert.Equal(t, "Name field is required when none of [Age,City] are present", v.Errors.One())
}

func TestVariadicArgs(t *testing.T) {
	// use custom validator
	v := New(M{
		"age": 2,
	})
	v.AddValidator("checkAge", func(_ DataFace, val any, ints ...int) bool {
		return Enum(val, ints)
	})
	v.StringRule("age", "required|checkAge:1,2,3,4")
	assert.True(t, v.Validate())

	v = New(M{
		"age": 2,
	})
	v.AddValidator("checkAge", func(_ DataFace, val any, ints ...any) bool {
		return Enum(val, ints)
	})
	v.StringRule("age", "required|checkAge:1,2,3,4")
	ok := v.Validate()
	assert.True(t, ok)
}

func TestValidation_Validate(t *testing.T) {
	v := New(M{
		"name": "haozi",
	})
	v.AddValidator("checkName", func(data DataFace, val any) bool {
		name, exist := data.Get("name")
		return name == "haozi" && exist
	})
	v.StringRule("name", "required|checkName")
	assert.True(t, v.Validate())

	v = New(M{
		"age": 2,
	})
	v.AddValidator("checkAge", func(data DataFace, val any) bool {
		age, exist := data.Get("age")
		return age == 2 && exist
	})
	v.StringRule("age", "required|checkAge")
	assert.True(t, v.Validate())
}

func TestValidation_Validate_filter(t *testing.T) {
	v := Map(M{
		"age": "abc",
	}).FilterRules(MS{
		"age": "int",
	})

	assert.False(t, v.Validate())
	assert.Equal(t, `age: strconv.Atoi: parsing "abc": invalid syntax`, v.Errors.One())
}

func TestValidation_Validate_argHasNil(t *testing.T) {
	// new rule
	r1 := NewRule("age", "eq", nil)

	d := M{
		"age": 23,
	}
	v := Map(d).AppendRules(r1)

	assert.False(t, v.Validate())
	assert.Equal(t, `age field did not pass validation`, v.Errors.One())
}

var nestedMap = map[string]any{
	"names": []string{"John", "Jane", "abc"},
	"coding": []map[string]any{
		{
			"details": map[string]any{
				"em": map[string]any{
					"code":              "001",
					"encounter_uid":     "1",
					"billing_provider":  "Test provider",
					"resident_provider": "Test Resident Provider",
				},
				"cpt": []map[string]any{
					{
						"code":              "001",
						"encounter_uid":     "1",
						"work_item_uid":     "1",
						"billing_provider":  "Test provider",
						"resident_provider": "Test Resident Provider",
					},
					{
						"code":              "OBS01",
						"encounter_uid":     "1",
						"work_item_uid":     "1",
						"billing_provider":  "Test provider",
						"resident_provider": "Test Resident Provider",
					},
					{
						"code":              "SU002",
						"encounter_uid":     "1",
						"work_item_uid":     "1",
						"billing_provider":  "Test provider",
						"resident_provider": "Test Resident Provider",
					},
				},
			},
		},
	},
}

func TestRequired_AllItemsPassed(t *testing.T) {
	v := Map(nestedMap)
	v.StopOnError = false
	v.StringRule("coding.*.details", "required")
	v.StringRule("coding.*.details.em", "required")
	v.StringRule("coding.*.details.cpt.*.encounter_uid", "required")
	v.StringRule("coding.*.details.cpt.*.work_item_uid", "required|string")
	assert.True(t, v.Validate())
	assert.Empty(t, v.Errors)
}

func TestRequired__map_subSlice_mDotStar(t *testing.T) {
	m := map[string]any{
		"names": []string{"John", "Jane", "abc"},
		"coding": []map[string]any{
			{
				"details": map[string]any{
					"em": map[string]any{
						"code":              "001",
						"encounter_uid":     "1",
						"billing_provider":  "Test provider",
						"resident_provider": "Test Resident Provider",
					},
					"cpt": []map[string]any{
						{
							"code": "001",
							// "encounter_uid":     "1",
							"work_item_uid":     "1",
							"billing_provider":  "Test provider",
							"resident_provider": "Test Resident Provider",
						},
						{
							"code":              "OBS01",
							"encounter_uid":     "2",
							"work_item_uid":     "1",
							"billing_provider":  "Test provider",
							"resident_provider": "Test Resident Provider",
						},
						{
							"code":              "SU002",
							"encounter_uid":     "3",
							"work_item_uid":     "1",
							"billing_provider":  "Test provider",
							"resident_provider": "Test Resident Provider",
						},
					},
				},
			},
		},
	}

	dump.Println(maputil.GetByPath("coding.*.details.cpt.*.encounter_uid", m))
	dump.Println(maputil.GetByPath("coding.*.details.cpt.*.work_item_uid", m))

	v := Map(m)
	v.StopOnError = false
	// v.StringRule("coding.*.details", "required")
	// v.StringRule("coding.*.details.em", "required")
	v.StringRule("coding.*.details.cpt.*.encounter_uid", "required")
	v.StringRule("coding.*.details.cpt.*.work_item_uid", "required")
	assert.False(t, v.Validate())
	dump.Println(v.Errors)
}

func TestValidate_map_subSlice_1dotStar(t *testing.T) {
	mp := map[string]any{
		"top": map[string]any{
			"cpt": []map[string]any{
				{
					"code": "001",
					// "encounter_uid":     "1",
					"work_item_uid":     "1",
					"billing_provider":  "Test provider",
					"resident_provider": "Test Resident Provider",
				},
				{
					"code":              "OBS01",
					"encounter_uid":     "2",
					"work_item_uid":     "1",
					"billing_provider":  "Test provider",
					"resident_provider": "Test Resident Provider",
				},
				{
					"code":              "SU002",
					"encounter_uid":     "3",
					"work_item_uid":     "1",
					"billing_provider":  "Test provider",
					"resident_provider": "Test Resident Provider",
				},
			},
		},
	}

	v := Map(mp)
	v.StopOnError = false
	v.StringRule("top.cpt.*.encounter_uid", "required")
	assert.False(t, v.Validate())

	s := v.Errors.String()
	assert.StrContains(t, s, "top.cpt.*.encounter_uid is required")

	// test invalid path
	v = Map(mp)
	v.StopOnError = false
	v.StringRule("top.*.cpt.*.invalid_path", "required")
	assert.False(t, v.Validate())
	// fmt.Println(v.Errors)
	assert.StrContains(t, v.Errors.String(), "required: top.*.cpt.*.invalid_path is required")
}

func TestRequired_MissingParentField(t *testing.T) {
	m := map[string]any{
		"names": []string{"John", "Jane", "abc"},
		"coding": []map[string]any{
			{
				"details": map[string]any{
					"em": map[string]any{
						"code":              "001",
						"encounter_uid":     "1",
						"billing_provider":  "Test provider",
						"resident_provider": "Test Resident Provider",
					},
				},
			},
		},
	}

	v := Map(m)
	v.StopOnError = false
	v.StringRule("coding.*.details", "required")
	v.StringRule("coding.*.details.em", "required")
	v.StringRule("coding.*.details.cpt.*.encounter_uid", "required")
	// not exist field
	v.StringRule("coding.*.details.cpt.*.not_exist_field", "required")
	es := v.ValidateE()
	assert.False(t, es.Empty())

	s := es.String()
	assert.StrContains(t, s, "coding.*.details.cpt.*.encounter_uid is required")
	assert.StrContains(t, s, "coding.*.details.cpt.*.not_exist_field is required")
}
