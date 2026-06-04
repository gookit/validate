package validate

import (
	"errors"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// builtinMeta fetches a builtin validator's funcMeta from the global registry.
// All targeted validators (min, minLength, isInt, requiredWith) are registered
// at init, so this always succeeds for them.
func builtinMeta(t *testing.T, name string) *funcMeta {
	t.Helper()
	fm, ok := validatorMetas[name]
	assert.Require(t, assert.True(t, ok, "validator %q must be a builtin", name))
	return fm
}

// TestConvertRuleArgs covers the pure build-time arg converter directly.
func TestConvertRuleArgs(t *testing.T) {
	t.Run("single any early-return keeps args", func(t *testing.T) {
		// min: func(val, minVal any) — second arg is interface, so convertRuleArgs
		// must leave the string arg untouched and return nil.
		fm := builtinMeta(t, "min")
		args := []any{"5"}
		err := convertRuleArgs(fm, "Age", args, 1)
		assert.NoErr(t, err)
		assert.Eq(t, "5", args[0]) // unchanged (still string)
	})

	t.Run("concrete int converts string to int", func(t *testing.T) {
		// minLength: func(val any, minLen int) — second arg is concrete int.
		fm := builtinMeta(t, "minLength")
		args := []any{"3"}
		err := convertRuleArgs(fm, "Name", args, 1)
		assert.NoErr(t, err)
		assert.Eq(t, 3, args[0]) // converted to int
	})

	t.Run("variadic converts each arg to elem type", func(t *testing.T) {
		// isInt: func(val any, minAndMax ...int64) — variadic int64.
		fm := builtinMeta(t, "isInt")
		args := []any{"3", "5"}
		err := convertRuleArgs(fm, "Age", args, 1)
		assert.NoErr(t, err)
		assert.Eq(t, int64(3), args[0])
		assert.Eq(t, int64(5), args[1])
	})

	t.Run("conversion failure returns argConvError", func(t *testing.T) {
		// minLength wants int, "abc" cannot convert -> *argConvError, args kept.
		fm := builtinMeta(t, "minLength")
		args := []any{"abc"}
		err := convertRuleArgs(fm, "Name", args, 1)
		assert.Err(t, err)

		var ce *argConvError
		assert.True(t, errors.As(err, &ce), "want *argConvError")
		assert.Eq(t, "Name", ce.field)
		assert.Eq(t, "abc", args[0]) // unchanged on failure
	})

	t.Run("empty args returns nil", func(t *testing.T) {
		// convertRuleArgs short-circuits on len(args)==0 before touching the
		// signature; any builtin meta works here.
		fm := builtinMeta(t, "minLength")
		assert.NoErr(t, convertRuleArgs(fm, "Name", nil, 1))
	})
}

// raStatic is a STATIC struct: minLength's int arg should be pre-converted at
// build time and the rule marked argsReady.
type raStatic struct {
	Name string `validate:"required|minLen:3"`
	Age  int    `validate:"min:1"`
}

// raDynamic is DYNAMIC (slice-of-struct): rules are built per value via
// parseRulesFromTag and must NOT be pre-converted (args stay string, argsReady
// stays false).
type raInner struct {
	Zip string `validate:"minLen:3"`
}

type raDynamic struct {
	Items []raInner `validate:"required"`
}

func findRule(v *Validation, field, realName string) *Rule {
	for _, r := range v.rules {
		if r.realName != realName {
			continue
		}
		for _, f := range r.fields {
			if f == field {
				return r
			}
		}
	}
	return nil
}

// TestArgsReadyBehavior checks the build-time pre-conversion打标 on real structs.
func TestArgsReadyBehavior(t *testing.T) {
	t.Run("static minLength pre-converted and argsReady", func(t *testing.T) {
		v := Struct(&raStatic{})

		r := findRule(v, "Name", "minLength")
		assert.Require(t, assert.NotNil(t, r))
		assert.True(t, r.argsReady, "static minLength rule must be argsReady")
		assert.Eq(t, 3, r.arguments[0]) // string -> int

		// min: func(val, minVal any) — single-any, args stay string but the rule
		// is still argsReady (runtime needs no conversion for it).
		rmin := findRule(v, "Age", "min")
		assert.Require(t, assert.NotNil(t, rmin))
		assert.True(t, rmin.argsReady, "single-any min rule is argsReady")
		assert.Eq(t, "1", rmin.arguments[0]) // unchanged string
	})

	t.Run("dynamic slice-of-struct keeps string, not argsReady", func(t *testing.T) {
		v := Struct(&raDynamic{Items: []raInner{{}}})

		r := findRule(v, "Items.0.Zip", "minLength")
		assert.Require(t, assert.NotNil(t, r))
		assert.False(t, r.argsReady, "dynamic minLength rule must NOT be argsReady")
		assert.Eq(t, "3", r.arguments[0]) // still string (runtime converts)
	})

	t.Run("static validation still passes/fails correctly", func(t *testing.T) {
		// sanity: pre-converted args produce correct validation outcomes.
		assert.True(t, Struct(&raStatic{Name: "abcd", Age: 5}).Validate())
		assert.False(t, Struct(&raStatic{Name: "ab", Age: 5}).Validate()) // minLen:3 fails
	})
}
