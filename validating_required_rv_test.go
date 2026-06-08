package validate

import (
	"testing"

	"github.com/gookit/goutil/x/assert"

	"github.com/gookit/validate/v2/internal/fieldval"
)

// TestRequiredByCtx_Parity asserts requiredByCtx (carrier RV-native) returns the
// same result as Required (public IsEmpty(any)) across map & struct sources and
// zero / non-zero values (RFC R4.2a §4). Both share the same *Validation state,
// so the only varying factor is the final IsEmpty source (carrier vs public).
func TestRequiredByCtx_Parity(t *testing.T) {
	cases := []struct {
		name  string
		field string
		val   any
	}{
		{"nonempty-string", "name", "tom"},
		{"empty-string", "name", ""},
		{"zero-int", "age", 0},
		{"nonzero-int", "age", 18},
		{"nil", "ext", nil},
		{"empty-slice", "tags", []string{}},
		{"nonempty-slice", "tags", []string{"a"}},
		{"false-bool", "ok", false},
		{"true-bool", "ok", true},
	}

	run := func(t *testing.T, mk func() *Validation) {
		for _, c := range cases {
			c := c
			t.Run(c.name, func(t *testing.T) {
				v := mk()
				want := v.Required(c.field, c.val)                              // public path
				got := v.requiredByCtx(c.field, fieldval.New(c.field, c.val))   // carrier path
				assert.Eq(t, want, got, "parity mismatch field=%s val=%#v", c.field, c.val)
			})
		}
	}

	t.Run("map-source", func(t *testing.T) {
		run(t, func() *Validation { return Map(map[string]any{"name": "x"}) })
	})

	t.Run("struct-source", func(t *testing.T) {
		type U struct {
			Name string
			Age  int
		}
		run(t, func() *Validation { return Struct(&U{Name: "x", Age: 1}) })
	})
}

// TestRequiredByCtx_RuleParity drives requiredByCtx through the real rule path
// (required validator) for map & struct sources and asserts pass/fail behavior is
// unchanged for zero / non-zero / missing fields.
func TestRequiredByCtx_RuleParity(t *testing.T) {
	t.Run("map-pass", func(t *testing.T) {
		v := New(map[string]any{"name": "tom", "age": 20})
		v.StringRule("name", "required")
		v.StringRule("age", "required")
		assert.True(t, v.Validate())
	})

	t.Run("map-fail-empty", func(t *testing.T) {
		v := New(map[string]any{"name": "", "age": 0})
		v.StringRule("name", "required")
		assert.False(t, v.Validate())
	})

	t.Run("struct-pass", func(t *testing.T) {
		type U struct {
			Name string `validate:"required"`
			Age  int    `validate:"required"`
		}
		v := Struct(&U{Name: "tom", Age: 1})
		assert.True(t, v.Validate())
	})

	t.Run("struct-fail-empty", func(t *testing.T) {
		type U struct {
			Name string `validate:"required"`
		}
		v := Struct(&U{Name: ""})
		assert.False(t, v.Validate())
	})
}
