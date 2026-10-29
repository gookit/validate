package validate_test

import (
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
