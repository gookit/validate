package validate

import (
	"sync"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// typeA: minLen on Name, email on Email. Name too short => fails.
type facTypeA struct {
	Name  string `validate:"required|minLen:8"`
	Email string `validate:"required|email"`
}

// typeB: different fields/rules entirely. Age out of range => fails.
type facTypeB struct {
	Title string `validate:"required|minLen:2"`
	Age   int    `validate:"required|int|min:1|max:120"`
}

// TestFactory_CrossContamination is the most important test: reusing a single
// Factory across different types must not leak any state (rules, errors,
// safeData, validators, scene, trans) from one validation into the next.
func TestFactory_CrossContamination(t *testing.T) {
	f := NewFactory()

	// --- round 1: type A, Name fails (too short), produces Errors + partial safeData ---
	a := facTypeA{Name: "bob", Email: "bob@example.com"}
	vA := f.Struct(&a)
	okA := vA.Validate()
	assert.False(t, okA)
	assert.True(t, vA.Errors.HasField("Name"))
	rulesCountA := len(vA.rules)
	assert.True(t, rulesCountA > 0)
	vA.Release()

	// After release the instance is fully reset.
	// --- round 2: SAME factory/pool, type B. Must be clean of all A residue. ---
	b := facTypeB{Title: "go", Age: 999}
	vB := f.Struct(&b)

	// pre-validate: no residual rules from A beyond B's own, no errors/safeData/scene
	assert.Len(t, vB.Errors, 0)
	assert.False(t, vB.hasError)
	assert.False(t, vB.hasValidated)
	assert.False(t, vB.hasFiltered)
	assert.Len(t, vB.safeData, 0)
	assert.Len(t, vB.filteredData, 0)
	assert.Eq(t, "", vB.scene)
	assert.Nil(t, vB.scenes)
	// B has its own 2 fields => the rules are B's, not A's. Assert no A fields.
	for _, r := range vB.rules {
		for _, fld := range r.fields {
			assert.True(t, fld != "Name" && fld != "Email",
				"leaked rule field from type A: %s", fld)
		}
	}
	// translator must not carry A's nothing-special; just assert empty custom msgs
	assert.Len(t, vB.trans.fieldMap, 0)
	assert.Len(t, vB.trans.labelMap, 0)

	okB := vB.Validate()
	assert.False(t, okB)
	assert.True(t, vB.Errors.HasField("Age"))
	// B's errors must not mention A's fields
	assert.False(t, vB.Errors.HasField("Name"))
	assert.False(t, vB.Errors.HasField("Email"))
	vB.Release()

	// --- round 3: type A again, must match the first A result exactly ---
	vA2 := f.Struct(&a)
	okA2 := vA2.Validate()
	assert.Eq(t, okA, okA2)
	assert.True(t, vA2.Errors.HasField("Name"))
	assert.Eq(t, rulesCountA, len(vA2.rules))
	vA2.Release()
}

// TestFactory_CustomValidatorNoLeak stresses the validators/validatorMetas leak
// path: UserForm registers a struct-local custom validator + custom messages +
// translates + ConfigValidation. After Release, a different type must not see
// any of those instance validators or translator entries.
func TestFactory_CustomValidatorNoLeak(t *testing.T) {
	f := NewFactory()

	u := &UserForm{Name: "inhere01", Email: "h@ex.com", Code: "1234"}
	vU := f.Struct(u)
	// disable StopOnError so the struct-local "customValidator" is actually
	// looked up (and binds into vU.validatorMetas) even though earlier fields
	// fail; this exercises the per-type validator-leak path on Release.
	vU.StopOnError = false
	vU.Validate()
	_, boundCustom := vU.validatorMetas["customValidator"]
	assert.True(t, boundCustom)
	// UserForm also populates translator custom data via Messages/Translates.
	assert.True(t, len(vU.trans.labelMap) > 0 || vU.trans.messages != nil)
	vU.Release()

	// Reuse for a plain type with no custom validators/messages.
	b := facTypeB{Title: "go", Age: 30}
	vB := f.Struct(&b)
	// custom validator from UserForm must be gone.
	_, hasCustom := vB.validatorMetas["customValidator"]
	assert.False(t, hasCustom)
	// translator custom data wiped.
	assert.Len(t, vB.trans.labelMap, 0)
	assert.Len(t, vB.trans.fieldMap, 0)
	assert.Nil(t, vB.trans.messages)
	vB.Release()
}

// TestFactory_StructEquivalence: factory result must equal validate.Struct().
func TestFactory_StructEquivalence(t *testing.T) {
	f := NewFactory()

	t.Run("typeA fail", func(t *testing.T) {
		a := facTypeA{Name: "bob", Email: "bad-email"}
		assertStructEquivalent(t, f, &a)
	})
	t.Run("typeA pass", func(t *testing.T) {
		a := facTypeA{Name: "inhere01", Email: "h@ex.com"}
		assertStructEquivalent(t, f, &a)
	})
	t.Run("typeB fail", func(t *testing.T) {
		b := facTypeB{Title: "g", Age: 999}
		assertStructEquivalent(t, f, &b)
	})
	t.Run("typeB pass", func(t *testing.T) {
		b := facTypeB{Title: "go", Age: 30}
		assertStructEquivalent(t, f, &b)
	})
	t.Run("flatUser pass", func(t *testing.T) {
		u := flatUser{Name: "inhere", Email: "john@example.com", Age: 30}
		assertStructEquivalent(t, f, &u)
	})
	t.Run("UserForm with custom msg", func(t *testing.T) {
		u := &UserForm{Name: "bob", Email: "h@ex.com", Code: "1234"}
		assertStructEquivalent(t, f, u)
	})
}

func assertStructEquivalent(t *testing.T, f *Factory, s any) {
	t.Helper()

	want := Struct(s)
	wantOK := want.Validate()

	got := f.Struct(s)
	gotOK := got.Validate()
	defer got.Release()

	assert.Eq(t, wantOK, gotOK)
	assert.Eq(t, len(want.Errors), len(got.Errors))
	// compare per-field error sets
	for field, wMsgs := range want.Errors {
		gMsgs, ok := got.Errors[field]
		assert.True(t, ok, "missing error field: %s", field)
		assert.Eq(t, len(wMsgs), len(gMsgs))
		for k, wm := range wMsgs {
			assert.Eq(t, wm, gMsgs[k])
		}
	}
}

// TestFactory_MapEquivalence: factory Map + StringRules must match validate.Map.
func TestFactory_MapEquivalence(t *testing.T) {
	f := NewFactory()

	build := func(create func() *Validation) (bool, Errors) {
		v := create()
		v.StringRules(MS{
			"name":  "required|minLen:3",
			"email": "required|email",
			"age":   "required|int|min:1|max:120",
		})
		ok := v.Validate()
		return ok, v.Errors
	}

	data := M{"name": "x", "email": "bad", "age": 999}

	wantOK, wantErrs := build(func() *Validation { return Map(map[string]any(data)) })
	gotV := f.Map(map[string]any(data))
	gotV.StringRules(MS{
		"name":  "required|minLen:3",
		"email": "required|email",
		"age":   "required|int|min:1|max:120",
	})
	gotOK := gotV.Validate()
	defer gotV.Release()

	assert.Eq(t, wantOK, gotOK)
	assert.Eq(t, len(wantErrs), len(gotV.Errors))
	for field := range wantErrs {
		_, ok := gotV.Errors[field]
		assert.True(t, ok, "missing error field: %s", field)
	}
}

// TestFactory_ReleaseNoOpDefault: default-path instances have pool==nil, so
// Release() is a harmless no-op.
func TestFactory_ReleaseNoOpDefault(t *testing.T) {
	v := Struct(&facTypeA{Name: "inhere01", Email: "h@ex.com"})
	assert.Nil(t, v.pool)
	v.Release() // must not panic
	v.Validate()
	assert.True(t, v.IsOK())
}

// TestFactory_Concurrent runs many goroutines each using the SAME factory; with
// -race this catches data sharing / contamination across pooled instances.
func TestFactory_Concurrent(t *testing.T) {
	f := NewFactory()

	var wg sync.WaitGroup
	for g := 0; g < 32; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				if g%2 == 0 {
					a := facTypeA{Name: "bob", Email: "bob@example.com"}
					v := f.Struct(&a)
					ok := v.Validate()
					assert.False(t, ok)
					assert.True(t, v.Errors.HasField("Name"))
					assert.False(t, v.Errors.HasField("Age"))
					v.Release()
				} else {
					b := facTypeB{Title: "go", Age: 999}
					v := f.Struct(&b)
					ok := v.Validate()
					assert.False(t, ok)
					assert.True(t, v.Errors.HasField("Age"))
					assert.False(t, v.Errors.HasField("Name"))
					v.Release()
				}
			}
		}(g)
	}
	wg.Wait()
}

// BenchmarkFactoryStructReuse mirrors BenchmarkStructFlat but reuses a pooled
// instance per iteration via the Factory. Compare allocs/op against
// BenchmarkStructFlat to see the pooling benefit.
func BenchmarkFactoryStructReuse(b *testing.B) {
	u := flatUser{Name: "inhere", Email: "john@example.com", Age: 30}
	f := NewFactory()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v := f.Struct(&u)
		v.Validate()
		v.Release()
	}
}
