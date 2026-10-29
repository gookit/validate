package validate_test

import (
	"encoding/json"
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2"
)

// TestInRule_sliceValue: the "in" rule accepts a slice/array value when every
// element is in the enum, and "not_in" is the inverse.
func TestInRule_sliceValue(t *testing.T) {
	type form struct {
		Tags []string `validate:"in:go,php"`
		Ids  []int    `validate:"in:1,2"`
	}

	tests := []struct {
		name string
		form *form
		want bool
	}{
		{"one element in", &form{Tags: []string{"go"}}, true},
		{"all elements in", &form{Tags: []string{"go", "php"}}, true},
		{"one element out", &form{Tags: []string{"go", "java"}}, false},
		{"all elements out", &form{Tags: []string{"java", "rust"}}, false},
		{"numeric all in", &form{Ids: []int{1, 2}}, true},
		{"numeric one out", &form{Ids: []int{1, 3}}, false},
		{"empty slice", &form{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := validate.Struct(tt.form)
			assert.Eq(t, tt.want, v.Validate())
		})
	}

	// the error keeps the rule message and the field path
	v := validate.Struct(&form{Tags: []string{"go", "java"}})
	assert.False(t, v.Validate())
	assert.NotEmpty(t, v.Errors.Field("Tags"))

	// not_in with a slice is the inverse
	type notInForm struct {
		Tags []string `validate:"not_in:go,php"`
	}
	assert.True(t, validate.Struct(&notInForm{Tags: []string{"java", "rust"}}).Validate())
	assert.False(t, validate.Struct(&notInForm{Tags: []string{"go", "java"}}).Validate())
}

func TestEnumInterfaceElements(t *testing.T) {
	goTag := "go"
	tests := []struct {
		name  string
		value any
		in    bool
		notIn bool
	}{
		{"allowed", []any{"go", "php"}, true, false},
		{"outside", []any{"java", "rust"}, false, true},
		{"mixed", []any{"go", "java"}, false, false},
		{"nil with allowed", []any{nil, "go"}, true, false},
		{"nil with outside", []any{nil, "java"}, false, true},
		{"pointer", []any{&goTag}, true, false},
		{"array", [2]any{"go", "php"}, true, false},
		{"unsupported boolean", []any{true}, false, false},
		{"unsupported map", []any{map[string]any{"tag": "go"}}, false, false},
		{"unsupported slice", []any{[]string{"go"}}, false, false},
		{"typed strings", []string{"go", "php"}, true, false},
		{"typed pointers", []*string{nil, &goTag}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Eq(t, tt.in, validate.Enum(tt.value, []string{"go", "php"}))
			assert.Eq(t, tt.notIn, validate.NotIn(tt.value, []string{"go", "php"}))
		})
	}
}

func TestEnumJSONArrays(t *testing.T) {
	for _, tt := range []struct {
		name string
		data string
		rule string
		want bool
	}{
		{"in allowed", `{"tags":["go","php"]}`, "in:go,php", true},
		{"in mixed", `{"tags":["go","java"]}`, "in:go,php", false},
		{"not in outside", `{"tags":["java","rust"]}`, "not_in:go,php", true},
		{"not in mixed", `{"tags":["go","java"]}`, "not_in:go,php", false},
		{"null element", `{"tags":[null,"go"]}`, "in:go,php", true},
		{"unsupported element", `{"tags":["go",false]}`, "in:go,php", false},
		{"numbers allowed", `{"tags":[1,2]}`, "in:1,2", true},
		{"number outside", `{"tags":[3,1]}`, "in:1,2", false},
		{"number with fraction", `{"tags":[1.5]}`, "in:1,2", false},
		{"numbers not in", `{"tags":[3,4]}`, "not_in:1,2", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var data map[string]any
			assert.NoErr(t, json.Unmarshal([]byte(tt.data), &data))
			v := validate.Map(data)
			v.StringRule("tags", tt.rule)
			assert.Eq(t, tt.want, v.Validate())
		})
	}
}
