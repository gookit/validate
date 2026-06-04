package validate

import (
	"database/sql"
	"reflect"
	"sync"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

// Money 自定义数值包装类型,用于验证数值校验器(min/max/between)对提取值生效。
type Money int64

// registerNullString 注册 sql.NullString 的提取器:Valid 时返回底层字符串,
// 否则返回 nil(视为空/未设置)。
func registerNullString() {
	AddCustomType(func(field reflect.Value) any {
		ns := field.Interface().(sql.NullString)
		if ns.Valid {
			return ns.String
		}
		return nil
	}, sql.NullString{})
}

// registerMoney 注册 Money 的提取器:返回其底层 int64。
func registerMoney() {
	AddCustomType(func(field reflect.Value) any {
		return int64(field.Interface().(Money))
	}, Money(0))
}

func TestAddCustomType_NullString(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()
	registerNullString()

	t.Run("required fails on invalid (extracted nil = empty)", func(t *testing.T) {
		// Valid:false with a non-empty String: raw struct is non-empty, only the
		// extractor turns it into nil -> proves extraction drives the result.
		v := Map(map[string]any{"name": sql.NullString{Valid: false, String: "x"}})
		v.StringRule("name", "required")
		assert.False(t, v.Validate())
		assert.True(t, v.Errors.HasField("name"))
	})

	t.Run("required passes on valid", func(t *testing.T) {
		v := Map(map[string]any{"name": sql.NullString{Valid: true, String: "x"}})
		v.StringRule("name", "required")
		assert.True(t, v.Validate())
	})

	t.Run("minLen fails on short extracted string", func(t *testing.T) {
		v := Map(map[string]any{"name": sql.NullString{Valid: true, String: "ab"}})
		v.StringRule("name", "minLen:3")
		assert.False(t, v.Validate())
	})

	t.Run("minLen passes on long enough extracted string", func(t *testing.T) {
		v := Map(map[string]any{"name": sql.NullString{Valid: true, String: "abc"}})
		v.StringRule("name", "minLen:3")
		assert.True(t, v.Validate())
	})
}

func TestAddCustomType_Money(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()
	registerMoney()

	t.Run("min on extracted int64", func(t *testing.T) {
		v := Map(map[string]any{"amount": Money(5)})
		v.StringRule("amount", "min:10")
		assert.False(t, v.Validate())

		v = Map(map[string]any{"amount": Money(10)})
		v.StringRule("amount", "min:10")
		assert.True(t, v.Validate())
	})

	t.Run("max on extracted int64", func(t *testing.T) {
		v := Map(map[string]any{"amount": Money(101)})
		v.StringRule("amount", "max:100")
		assert.False(t, v.Validate())

		v = Map(map[string]any{"amount": Money(100)})
		v.StringRule("amount", "max:100")
		assert.True(t, v.Validate())
	})

	t.Run("between on extracted int64", func(t *testing.T) {
		v := Map(map[string]any{"amount": Money(50)})
		v.StringRule("amount", "between:10,100")
		assert.True(t, v.Validate())

		v = Map(map[string]any{"amount": Money(9)})
		v.StringRule("amount", "between:10,100")
		assert.False(t, v.Validate())

		v = Map(map[string]any{"amount": Money(200)})
		v.StringRule("amount", "between:10,100")
		assert.False(t, v.Validate())
	})
}

func TestAddCustomType_Struct(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()
	registerNullString()
	registerMoney()

	type Order struct {
		Title  sql.NullString `validate:"required|minLen:3"`
		Amount Money          `validate:"min:10|max:100"`
	}

	t.Run("valid struct passes", func(t *testing.T) {
		o := &Order{
			Title:  sql.NullString{Valid: true, String: "book"},
			Amount: Money(50),
		}
		assert.True(t, Struct(o).Validate())
	})

	t.Run("invalid title (nil) fails required", func(t *testing.T) {
		o := &Order{
			Title:  sql.NullString{Valid: false},
			Amount: Money(50),
		}
		v := Struct(o)
		assert.False(t, v.Validate())
		assert.True(t, v.Errors.HasField("Title"))
	})

	t.Run("amount out of range fails", func(t *testing.T) {
		o := &Order{
			Title:  sql.NullString{Valid: true, String: "book"},
			Amount: Money(5),
		}
		v := Struct(o)
		assert.False(t, v.Validate())
		assert.True(t, v.Errors.HasField("Amount"))
	})
}

func TestAddCustomType_UnregisteredTypeUnchanged(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()
	// only register Money, not NullString
	registerMoney()

	t.Run("plain string behavior unchanged", func(t *testing.T) {
		v := Map(map[string]any{"name": "ab"})
		v.StringRule("name", "minLen:3")
		assert.False(t, v.Validate())

		v = Map(map[string]any{"name": "abc"})
		v.StringRule("name", "minLen:3")
		assert.True(t, v.Validate())
	})

	t.Run("plain int behavior unchanged", func(t *testing.T) {
		v := Map(map[string]any{"age": 5})
		v.StringRule("age", "min:10")
		assert.False(t, v.Validate())

		v = Map(map[string]any{"age": 20})
		v.StringRule("age", "min:10")
		assert.True(t, v.Validate())
	})
}

func TestResetCustomTypes(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()
	registerNullString()

	// Use a discriminating value: Valid:false but a non-empty String field, so the
	// RAW struct is non-empty (passes required) while the EXTRACTOR returns nil.
	// This isolates the extraction effect from IsEmpty's zero-struct handling.
	bad := sql.NullString{Valid: false, String: "x"}

	// before reset: extraction yields nil -> required fails.
	v := Map(map[string]any{"name": bad})
	v.StringRule("name", "required")
	assert.False(t, v.Validate())

	ResetCustomTypes()
	assert.False(t, hasCustomTypes.Load())

	// after reset: no extraction. the non-empty raw struct value passes required.
	v = Map(map[string]any{"name": bad})
	v.StringRule("name", "required")
	assert.True(t, v.Validate())
}

func TestResolveCustomType_Internal(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()

	t.Run("gate off returns original", func(t *testing.T) {
		got, ok := resolveCustomType("hello")
		assert.False(t, ok)
		assert.Eq(t, "hello", got)
	})

	t.Run("nil val is safe", func(t *testing.T) {
		registerNullString()
		got, ok := resolveCustomType(nil)
		assert.False(t, ok)
		assert.Nil(t, got)
	})

	t.Run("registered hit extracts", func(t *testing.T) {
		registerNullString()
		got, ok := resolveCustomType(sql.NullString{Valid: true, String: "v"})
		assert.True(t, ok)
		assert.Eq(t, "v", got)
	})

	t.Run("unregistered miss returns original", func(t *testing.T) {
		registerNullString()
		got, ok := resolveCustomType(Money(7))
		assert.False(t, ok)
		assert.Eq(t, Money(7), got)
	})
}

func TestAddCustomType_NilGuards(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()

	// nil fn -> no-op, gate stays off
	AddCustomType(nil, sql.NullString{})
	assert.False(t, hasCustomTypes.Load())

	// no types -> no-op, gate stays off
	AddCustomType(func(reflect.Value) any { return nil })
	assert.False(t, hasCustomTypes.Load())

	// nil sample mixed with a valid one: nil is skipped, valid type still registered.
	AddCustomType(func(field reflect.Value) any {
		return field.Interface().(sql.NullString).String
	}, nil, sql.NullString{})
	assert.True(t, hasCustomTypes.Load())
	// the nil sample must not have been stored under the nil/invalid type.
	_, niStored := customTypes.Load(reflect.TypeOf(nil))
	assert.False(t, niStored)
	// the valid sample IS stored and drives extraction.
	got, ok := resolveCustomType(sql.NullString{Valid: true, String: "hi"})
	assert.True(t, ok)
	assert.Eq(t, "hi", got)
}

// TestAddCustomType_WildcardSlice verifies that custom-type extraction also
// applies to each element of a ".*" wildcard slice path.
func TestAddCustomType_WildcardSlice(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()
	registerMoney()
	registerNullString()

	t.Run("numeric element extraction (min)", func(t *testing.T) {
		v := Map(map[string]any{"prices": []Money{Money(20), Money(30)}})
		v.StringRule("prices.*", "min:10")
		assert.True(t, v.Validate())

		v = Map(map[string]any{"prices": []Money{Money(20), Money(5)}})
		v.StringRule("prices.*", "min:10")
		assert.False(t, v.Validate())
	})

	t.Run("string element extraction (minLen)", func(t *testing.T) {
		v := Map(map[string]any{"names": []sql.NullString{
			{Valid: true, String: "abcd"},
			{Valid: true, String: "efgh"},
		}})
		v.StringRule("names.*", "minLen:3")
		assert.True(t, v.Validate())

		v = Map(map[string]any{"names": []sql.NullString{
			{Valid: true, String: "abcd"},
			{Valid: true, String: "xy"},
		}})
		v.StringRule("names.*", "minLen:3")
		assert.False(t, v.Validate())
	})

	t.Run("element extracted to nil fails required", func(t *testing.T) {
		v := Map(map[string]any{"names": []sql.NullString{
			{Valid: true, String: "abc"},
			{Valid: false, String: "x"}, // extracts to nil
		}})
		v.StringRule("names.*", "required")
		assert.False(t, v.Validate())
	})
}

// TestAddCustomType_Concurrent exercises concurrent register + validate under -race.
func TestAddCustomType_Concurrent(t *testing.T) {
	ResetCustomTypes()
	defer ResetCustomTypes()

	var wg sync.WaitGroup
	// concurrent registrations
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registerNullString()
			registerMoney()
		}()
	}
	// concurrent validations interleaved with registration
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := Map(map[string]any{
				"name":   sql.NullString{Valid: true, String: "abc"},
				"amount": Money(50),
			})
			v.StringRule("name", "minLen:3")
			v.StringRule("amount", "between:10,100")
			_ = v.Validate()
		}()
	}
	wg.Wait()
}
