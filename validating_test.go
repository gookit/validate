package validate

import (
	"testing"

	"github.com/gookit/goutil/dump"
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
	v := New(M{
		"name": "lee",
		"age":  "12",
	})
	v.StringRules(MS{
		"age":     "required_if:name,lee",
		"nothing": "required_if:age,12,13,14",
	})

	v.Validate()
	assert.Equal(t, "nothing is required when age is [12,13,14]", v.Errors.One())
}

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

	v.Validate()
	assert.Equal(t, "nothing field is required unless age is in [12,13,14]", v.Errors.One())
}

func TestValidation_RequiredWith(t *testing.T) {
	v := New(M{
		"age":  "18",
		"name": "test",
	})
	v.StringRules(MS{
		"age":      "required_with:name,city",
		"anything": "required_with:sex,city",
		"nothing":  "required_with:age,name",
	})

	v.Validate()
	assert.Equal(t, "nothing field is required when [age,name] is present", v.Errors.One())
}

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

	v.Validate()
	// fmt.Println(v.Errors)
	assert.Equal(t, "nothing field is required when [age,name,sex] is present", v.Errors.One())
}

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

	v.Validate()
	assert.Equal(t, "nothing field is required when [sex,name] is not present", v.Errors.One())
}

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

	v.Validate()
	assert.Equal(t, "nothing field is required when none of [sex,city] are present", v.Errors.One())
}

func TestVariadicArgs(t *testing.T) {
	// use custom validator
	v := New(M{
		"age": 2,
	})
	v.AddValidator("checkAge", func(val any, ints ...int) bool {
		return Enum(val, ints)
	})
	v.StringRule("age", "required|checkAge:1,2,3,4")
	assert.True(t, v.Validate())

	v = New(M{
		"age": 2,
	})
	v.AddValidator("checkAge", func(val any, ints ...any) bool {
		return Enum(val, ints)
	})
	v.StringRule("age", "required|checkAge:1,2,3,4")
	ok := v.Validate()
	assert.True(t, ok)
}

func TestValidation_Validate_filter(t *testing.T) {
	v := Map(M{
		"age": "abc",
	}).FilterRules(MS{
		"age": "int",
	})

	assert.False(t, v.Validate())
	assert.Equal(t, `strconv.Atoi: parsing "abc": invalid syntax`, v.Errors.One())
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
	// v.StringRule("coding.*.details", "required")
	v.StringRule("coding.*.details.em", "required")
	v.StringRule("coding.*.details.cpt.*.encounter_uid", "required")
	v.StringRule("coding.*.details.cpt.*.work_item_uid", "required")
	assert.True(t, v.Validate())
	// fmt.Println(v.Errors)
}

func TestRequired_MissingField(t *testing.T) {
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

	v := Map(m)
	v.StopOnError = false
	// v.StringRule("coding.*.details", "required")
	// v.StringRule("coding.*.details.em", "required")
	v.StringRule("coding.*.details.cpt.*.encounter_uid", "required")
	// v.StringRule("coding.*.details.cpt.*.work_item_uid", "required")
	// assert.False(t, v.Validate()) // TODO ... fix this
	dump.Println(v.Errors)
}

func TestValidate_sliceValue_1dotStar(t *testing.T) {
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
	// v.StringRule("coding.*.details", "required")
	// v.StringRule("coding.*.details.em", "required")
	v.StringRule("coding.*.details.cpt.*.encounter_uid", "required")
	v.StringRule("coding.*.details.cpt.*.work_item_uid", "required")
	es := v.ValidateE()
	assert.False(t, es.Empty())
	s := es.String()
	assert.StrContains(t, s, "coding.*.details.cpt.*.encounter_uid is required")
	assert.StrContains(t, s, "coding.*.details.cpt.*.work_item_uid is required")
}
