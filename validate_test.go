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
	assert.Equal(t, v.Errors.One(), "nothing is required when age is [12 13 14]")
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
	assert.Equal(t, v.Errors.One(), "nothing field is required unless age is in [12 13 14]")
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
	assert.Equal(t, v.Errors.One(), "nothing field is required when [age name] is present")
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
	assert.Equal(t, v.Errors.One(), "nothing field is required when [age name sex] is present")
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
	assert.Equal(t, v.Errors.One(), "nothing field is required when [sex name] is not present")
}

func TestValidation_RequiredWithoutAll(t *testing.T) {
	v := New(M{
		"age":     "18",
		"name":    "test",
		"nothing": "",
	})
	v.StringRules(MS{
		"age":      "required_without_all:name,city",
		"anything": "required_without:age,name",
		"nothing":  "required_without_all:sex,city",
	})

	v.Validate()
	assert.Equal(t, v.Errors.One(), "nothing field is required when none of [sex city] are present")
}
