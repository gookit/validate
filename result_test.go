package validate

import (
	"sync"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

type checkUser struct {
	Name  string `validate:"required|min_len:3" json:"name"`
	Email string `validate:"required|email" json:"email"`
	Age   int    `validate:"required|min:1|max:120" json:"age"`
}

func validCheckUser() *checkUser {
	return &checkUser{Name: "inhere", Email: "test@example.com", Age: 30}
}

func invalidCheckUser() *checkUser {
	return &checkUser{Name: "x", Email: "not-an-email", Age: 999}
}

// Check on valid struct: ok face + safe data + bind.
func TestCheck_valid(t *testing.T) {
	r := Check(validCheckUser())

	assert.True(t, r.IsOK())
	assert.False(t, r.Fail())
	assert.NoErr(t, r.Err())
	assert.True(t, r.Errors.Empty())

	// safe data keyed by struct field name
	assert.NotEmpty(t, r.SafeData())
	assert.Eq(t, "inhere", r.SafeVal("Name"))
	val, ok := r.Safe("Age")
	assert.True(t, ok)
	assert.Eq(t, 30, val)

	// bind back to a struct (json keys, case-insensitive match)
	var out checkUser
	assert.NoErr(t, r.BindStruct(&out))
	assert.Eq(t, "inhere", out.Name)
	assert.Eq(t, 30, out.Age)
}

// Check on invalid struct: fail face + empty safe data.
func TestCheck_invalid(t *testing.T) {
	r := Check(invalidCheckUser())

	assert.False(t, r.IsOK())
	assert.True(t, r.Fail())
	assert.Err(t, r.Err())
	assert.False(t, r.Errors.Empty())

	// safe data cleared on error
	assert.Empty(t, r.SafeData())

	// BindStruct is a no-op (no safe data) and returns nil
	var out checkUser
	assert.NoErr(t, r.BindStruct(&out))
	assert.Eq(t, "", out.Name)
}

// ValidateR is the primitive: works on any configured instance (struct).
func TestValidateR_struct(t *testing.T) {
	r := Struct(validCheckUser()).ValidateR()
	assert.True(t, r.IsOK())
	assert.Eq(t, "inhere", r.SafeVal("Name"))

	r = Struct(invalidCheckUser()).ValidateR()
	assert.True(t, r.Fail())
}

// ValidateR on a map + programmatic rules (the documented map path).
func TestValidateR_map(t *testing.T) {
	v := Map(map[string]any{"age": 100, "name": "inhere"})
	v.StringRules(MS{
		"name": "required|string",
		"age":  "required|int|min:1",
	})

	r := v.ValidateR()
	assert.True(t, r.IsOK())
	assert.Eq(t, 100, r.SafeVal("age"))
	assert.Eq(t, "inhere", r.SafeVal("name"))
}

// ValidateR with a scene arg routes through Validate(scene...).
func TestValidateR_scene(t *testing.T) {
	v := Map(map[string]any{"name": "ab"})
	v.StringRule("name", "required|minLen:5")
	v.WithScenes(SValues{"s1": []string{"name"}})

	// scene s1 checks name -> fails
	r := v.ValidateR("s1")
	assert.True(t, r.Fail())
}

// The returned result is a snapshot, independent of later pooled reuse.
func TestCheck_resultIndependentAfterReuse(t *testing.T) {
	r1 := Check(validCheckUser())
	assert.True(t, r1.IsOK())
	name := r1.SafeVal("Name")

	// hammer the pool with other validations (valid + invalid mixed)
	for i := 0; i < 50; i++ {
		_ = Check(invalidCheckUser())
		_ = Check(&checkUser{Name: "other", Email: "o@e.com", Age: 10})
	}

	// r1 must be unchanged by the reuse
	assert.True(t, r1.IsOK())
	assert.Eq(t, name, r1.SafeVal("Name"))
	assert.Eq(t, "inhere", r1.SafeVal("Name"))
}

// Repeated Check calls reuse pooled instances but yield independent, correct
// results with no cross-contamination.
func TestCheck_poolReuse(t *testing.T) {
	for i := 0; i < 100; i++ {
		rok := Check(validCheckUser())
		assert.True(t, rok.IsOK())
		assert.True(t, rok.Errors.Empty())

		rbad := Check(invalidCheckUser())
		assert.True(t, rbad.Fail())
		assert.False(t, rbad.Errors.Empty())
	}
}

type checkAddr struct {
	City string `validate:"required|min_len:2" json:"city"`
	Zip  string `validate:"required" json:"zip"`
}

// S3: the pooled StructData (v.sd) is reused across calls — including across
// DIFFERENT struct types. Verify no field-name bleed / source leak between types.
func TestCheck_poolReuse_mixedTypes(t *testing.T) {
	for i := 0; i < 50; i++ {
		ru := Check(validCheckUser()) // type checkUser
		assert.True(t, ru.IsOK())
		assert.Eq(t, "inhere", ru.SafeVal("Name"))

		ra := Check(&checkAddr{City: "NYC", Zip: "10001"}) // different type, reused pool slot
		assert.True(t, ra.IsOK())
		assert.Eq(t, "NYC", ra.SafeVal("City"))
		// checkUser's fields must NOT leak into the addr result
		_, hasName := ra.Safe("Name")
		assert.False(t, hasName)

		rbad := Check(&checkAddr{City: "x"}) // City too short + Zip missing
		assert.True(t, rbad.Fail())
	}
}

// Concurrency: Check of MIXED types from many goroutines must be race-clean and
// correct (each goroutine gets its own pooled instance + StructData).
func TestCheck_concurrent_mixedTypes(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if r := Check(validCheckUser()); !r.IsOK() || r.SafeVal("Name") != "inhere" {
					t.Errorf("user: unexpected result: ok=%v name=%v", r.IsOK(), r.SafeVal("Name"))
				}
				if r := Check(&checkAddr{City: "NYC", Zip: "10001"}); !r.IsOK() || r.SafeVal("City") != "NYC" {
					t.Errorf("addr: unexpected result: ok=%v city=%v", r.IsOK(), r.SafeVal("City"))
				}
			}
		}()
	}
	wg.Wait()
}

// Concurrency: Check must be safe from many goroutines (run with -race).
func TestCheck_concurrent(t *testing.T) {
	var wg sync.WaitGroup
	for g := 0; g < 16; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if r := Check(validCheckUser()); !r.IsOK() {
					t.Errorf("expected ok, got errors: %v", r.Errors)
				}
				if r := Check(invalidCheckUser()); !r.Fail() {
					t.Errorf("expected fail, got ok")
				}
			}
		}()
	}
	wg.Wait()
}

// Filtered data is carried over into the result.
func TestValidResult_filtered(t *testing.T) {
	v := Map(map[string]any{"age": "100", "name": "inhere"})
	v.FilterRule("age", "int")
	v.StringRules(MS{
		"name": "required|string",
		"age":  "required|int|min:1",
	})

	r := v.ValidateR()
	assert.True(t, r.IsOK())
	assert.Eq(t, 100, r.Filtered("age"))
	assert.NotEmpty(t, r.FilteredData())
}
