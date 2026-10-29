package validate_test

import (
	"testing"

	"github.com/gookit/goutil/x/assert"
	"github.com/gookit/validate/v2"
)

type conditionalStatus string
type conditionalCount int
type conditionalByte uint8
type conditionalEnabled bool
type conditionalRatio float32

func TestRequiredIfDefinedTypes(t *testing.T) {
	member := conditionalStatus("member")
	memberPtr := &member
	var absentStatus *conditionalStatus
	var absentName *string
	emptyName := ""
	presentName := "Ada"
	for _, tt := range []struct {
		name  string
		mode  any
		rule  string
		value any
		want  bool
	}{
		{"defined string", member, "required_if:mode,member", "", false},
		{"defined string pointer", &member, "required_if:mode,member", "", false},
		{"defined string double pointer", &memberPtr, "required_if:mode,member", "", false},
		{"nil defined pointer", absentStatus, "required_if:mode,member", "", true},
		{"nil source pointer", member, "required_if:mode,member", absentName, false},
		{"empty source pointer", member, "required_if:mode,member", &emptyName, false},
		{"present source pointer", member, "required_if:mode,member", &presentName, true},
		{"defined integer", conditionalCount(2), "required_if:mode,2", "", false},
		{"defined unsigned", conditionalByte(2), "required_if:mode,2", "", false},
		{"defined boolean", conditionalEnabled(true), "required_if:mode,true", "", false},
		{"defined false boolean", conditionalEnabled(false), "required_if:mode,false", "", false},
		{"defined float", conditionalRatio(1.5), "required_if:mode,1.5", "", false},
		{"nonmatching string", conditionalStatus("guest"), "required_if:mode,member", "", true},
		{"matching with value", member, "required_if:mode,member", "Ada", true},
		{"multiple literals", member, "required_if:mode,member,guest", "", false},
		{"builtin string", "member", "required_if:mode,member", "", false},
		{"invalid literal", conditionalCount(2), "required_if:mode,no-number", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			v := validate.Map(map[string]any{"mode": tt.mode, "name": tt.value})
			v.StringRule("name", tt.rule)
			assert.Eq(t, tt.want, v.Validate())
		})
	}
}

func TestRequiredIfDefinedTypeStructField(t *testing.T) {
	type account struct {
		Mode conditionalStatus
		Name string `validate:"required_if:Mode,member"`
	}
	v := validate.Struct(&account{Mode: "member"})
	assert.False(t, v.Validate())
	assert.NotEmpty(t, v.Errors.Field("Name"))
	assert.True(t, validate.Struct(&account{Mode: "guest"}).Validate())
	assert.True(t, validate.Struct(&account{Mode: "member", Name: "Ada"}).Validate())
}
