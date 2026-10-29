package validate_test

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2"
)

// TestEnumRule_floatValue: a JSON number decodes into a float64, so the enum rules compare
// the integral value it holds. A fractional number cannot be compared with an int list.
func TestEnumRule_floatValue(t *testing.T) {
	cases := []struct {
		name  string
		value float64
		rule  string
		want  bool
	}{
		{"in, integral", 1, "in:1,2", true},
		{"in, outside", 3, "in:1,2", false},
		{"in, fraction", 1.5, "in:1,2", false},
		{"not_in, integral", 1, "not_in:1,2", false},
		{"not_in, outside", 3, "not_in:1,2", true},
		{"not_in, fraction", 1.5, "not_in:1,2", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := validate.Map(map[string]any{"scale": c.value})
			v.StringRule("scale", c.rule)
			assert.Eq(t, c.want, v.Validate())
		})
	}

	// a struct float field goes through the same conversion
	type form struct {
		Scale float64 `validate:"in:1,2"`
	}
	assert.True(t, validate.Struct(&form{Scale: 1}).Validate())
	assert.True(t, validate.Struct(&form{Scale: 2}).Validate())
	assert.False(t, validate.Struct(&form{Scale: 2.5}).Validate())

	// a string value keeps comparing as a string, not as a number
	v := validate.Map(map[string]any{"scale": "1.0"})
	v.StringRule("scale", "in:1,2")
	assert.False(t, v.Validate())
}

// TestEnumRule_mapElement: an unsupported map inside a slice reports the type error, the
// same way a map field does, instead of "value must be in the enum [...]".
func TestEnumRule_mapElement(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value any
		rule  string
	}{
		{"in", []any{map[string]any{"tag": "go"}}, "in:go,php"},
		{"not_in", []any{map[string]any{"tag": "go"}}, "not_in:go,php"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := validate.Map(map[string]any{"Tags": tt.value})
			v.StringRule("Tags", tt.rule)
			assert.False(t, v.Validate())
			assert.StrContains(t, v.Errors.FieldOne("Tags"), "does not support a map value")
		})
	}
}
