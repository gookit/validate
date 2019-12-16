package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestStructUseDefault(t *testing.T) {
	is := assert.New(t)

	type user struct {
		Name string `validate:"required|default:tom" filter:"trim|upper"`
		Age  int
	}

	u := &user{Age: 90}
	v := New(u)
	is.True(v.Validate())
	is.Equal("tom", u.Name)

	// check/filter default value
	u = &user{Age: 90}
	v = New(u)
	v.CheckDefault = true

	is.True(v.Validate())
	is.Equal("TOM", u.Name)
}

func TestValidation_RequiredIf(t *testing.T) {
	v := New(M{
		"age":     "12",
		"nothing": "",
	})
	v.StringRules(MS{
		"nothing": "required_if:age,12,13,14",
	})

	v.Validate()
	assert.Equal(t, v.Errors.One(), "nothing is required when age is [12 13 14]")
}

func TestValidation_RequiredUnless(t *testing.T) {
	v := New(M{
		"age":     "18",
		"nothing": "",
	})
	v.StringRules(MS{
		"nothing": "required_unless:age,12,13,14",
	})

	v.Validate()
	assert.Equal(t, v.Errors.One(), "nothing field is required unless age is in [12 13 14]")
}

func TestValidation_RequiredWith(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "test",
		"nothing": "",
	})
	v.StringRules(MS{
		"nothing": "required_with:age,name",
	})

	v.Validate()
	assert.Equal(t, v.Errors.One(), "nothing field is required when [age name] is present")
}

func TestValidation_RequiredWithAll(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "test",
		"nothing": "",
	})
	v.StringRules(MS{
		"nothing": "required_with:age,name,sex",
	})

	v.Validate()
	assert.Equal(t, v.Errors.One(), "nothing field is required when [age name sex] is present")
}


func TestValidation_RequiredWithout(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "test",
		"nothing": "",
	})
	v.StringRules(MS{
		"nothing": "required_with:big,sex",
	})

	v.Validate()
	assert.Equal(t, v.Errors.One(), "nothing field is required when [big sex] is present")
}

func TestValidation_RequiredWithoutAll(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "test",
		"nothing": "",
	})
	v.StringRules(MS{
		"nothing": "required_with:big,age",
	})

	v.Validate()
	assert.Equal(t, v.Errors.One(), "nothing field is required when [big age] is present")
}
