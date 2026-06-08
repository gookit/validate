package validate

import (
	"errors"
	"net/url"
	"sync"
	"testing"

	"github.com/gookit/filter"
	"github.com/gookit/goutil/x/assert"
)

func TestRule_basic(t *testing.T) {
	is := assert.New(t)
	data := url.Values{
		"name": []string{"inhere"},
		"age":  []string{"10"},
		"key0": []string{"val0"},
	}

	v := New(data)
	// new rule
	r := NewRule("name", "minLen", 6)
	r.SetScene("test") // only validate on scene "test"
	r.SetFilterFunc(func(val any) (any, error) {
		return val.(string) + "-HI", nil
	})
	r.SetBeforeFunc(func(_ *Validation) bool {
		return true
	})

	is.Equal([]string{"name"}, r.Fields())
	v.AppendRule(r)
	v.AddRule("field0", "required").SetOptional(true)
	v.AddRule("key0", "inRule").SetCheckFunc(func(s string) bool {
		return s == "val0"
	})
	v.AddRule("name", "ltField", "key0")

	// validate. will skip validate field "name"
	vr := v.ValidateR()
	is.True(vr.IsOK())
	is.Empty(vr.Errors)
	is.Equal("val0", vr.SafeVal("key0"))
	is.Equal(nil, vr.SafeVal("not-exist"))

	// validate on "test". will validate field "name"
	v.ResetResult()
	vr = v.ValidateR("test")
	is.True(vr.IsOK())
	is.Equal("val0", vr.SafeVal("key0"))
	is.Equal("inhere-HI", vr.SafeVal("name"))
}

func TestRule_SetBeforeFunc(t *testing.T) {
	is := assert.New(t)
	mp := M{
		"name":   "inhere",
		"avatar": "/some/file",
	}

	v := Map(mp)
	v.AddRule("avatar", "isFile")
	is.False(v.Validate())
	is.Equal("avatar must be an uploaded file", v.Errors.One())

	// use SetBeforeFunc
	v = Map(mp)
	v.AddRule("avatar", "isFile").
		SetBeforeFunc(func(_ *Validation) bool {
			// return false for skip validate
			return false
		})

	v.Validate()
	is.True(v.IsOK())
}

func TestRule_SetFilterFunc(t *testing.T) {
	is := assert.New(t)
	v := Map(M{
		"name": "inhere",
		"age":  "abc",
	})

	v.AddRule("age", "int", 1, 100).
		SetFilterFunc(func(val any) (i any, e error) {
			return filter.Int(val)
		})

	is.False(v.Validate())
	is.Equal(`age: strconv.Atoi: parsing "abc": invalid syntax`, v.Errors.One())
}

func TestRule_SetSkipEmpty(t *testing.T) {
	is := assert.New(t)
	mp := M{
		"name": "inhere",
		"age":  0,
	}

	v := Map(mp)
	v.AddRule("age", "int", 1)
	v.AddRule("name", "string", 1, 10)
	r := v.ValidateR()
	is.True(r.IsOK())

	sd := r.SafeData()
	is.Contains(sd, "name")
	is.NotContains(sd, "age")
	is.Equal("inhere", r.SafeVal("name"))
	is.Equal(nil, r.SafeVal("age"))

	v = Map(mp)
	v.AddRule("age", "int", 1).SetSkipEmpty(false)
	v.AddRule("name", "string", 1, 10)
	is.False(v.Validate())
	is.Equal("age value must be an integer and mix value is 1", v.Errors.One())
}

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

// TestSharedArgs_concurrentErrorFormat stress-tests the P3b shared-args path:
// many goroutines validate the SAME static type and all FAIL, forcing error
// message formatting (which reads the shared args). With the format() copy-on-
// write fix this must be race-free (run with -race). Use a struct whose minLen
// arg ("3") collides with a registered field label name to also exercise the
// label-substitution branch in Translator.format against shared args.
func TestSharedArgs_concurrentErrorFormat(t *testing.T) {
	defer ResetTypeCache()
	ResetTypeCache()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := Struct(&raStatic{Name: "x", Age: 0}) // both rules fail
			ok := v.Validate()
			_ = v.Errors.String()
			assert.False(t, ok)
		}()
	}
	wg.Wait()
}

// TestSharedRules_concurrentValidate verifies that sharing the immutable
// argsReady template *Rule pointer across instances (instantiateStatic no longer
// clones them) is race-free and produces consistent results: many goroutines
// validate the SAME static type concurrently. Run with -race.
func TestSharedRules_concurrentValidate(t *testing.T) {
	defer ResetTypeCache()
	ResetTypeCache()

	const n = 100
	results := make([]bool, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			// Name="abcd" (minLen:3 OK), Age=5 (min OK) -> all rules pass.
			results[idx] = Struct(&raStatic{Name: "abcd", Age: 5}).Validate()
		}(i)
	}
	wg.Wait()

	// all goroutines must agree the value is valid; shared template rules stay clean.
	for i := 0; i < n; i++ {
		assert.True(t, results[i], "goroutine %d must report valid", i)
	}

	// a fresh validation of an invalid value still fails -> template not polluted.
	assert.False(t, Struct(&raStatic{Name: "ab", Age: 0}).Validate())
}
