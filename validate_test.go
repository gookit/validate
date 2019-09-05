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

func TestDefaultOption(t *testing.T) {
	tests := []struct {
		name string
		json string
		res  []string
	}{
		{"empty", `{}`, []string{"required"}},
		{"zero set", `{"a":0}`, []string{"min value"}},
		{"valid value", `{"a":1}`, nil},
		{"last valid error value", `{"a":1000}`, []string{"max value"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v, err := FromJSON(test.json)
			assert.NoError(t, err)
			vld := v.Create()
			// desired behaviour:
			// if not set => required
			// if was provider default int value == 0 => report about min value requirement
			vld.StringRules(MS{
				"a": "min:1|required|max:999",
			})
			vld.CheckDefault = true
			vld.SkipOnEmpty = true
			vld.StopOnError = true
			_ = vld.Validate()
			assert.Len(t, vld.Errors, len(test.res))
			for i := range test.res {
				assert.Contains(t, vld.Errors.String(), test.res[i])
			}
		})
	}
}

func TestCheckDefaultOption(t *testing.T) {

	v := New(map[string][]string{
		"age":  {"12"},
		"name": {"inhere"},
	})
	v.CheckDefault = true
	v.SkipOnEmpty = true
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
