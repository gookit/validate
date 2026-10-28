package validate

import (
	"fmt"
	"math"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

type structMapStringKey string
type structMapIntKey int

type structMapEntry struct {
	Name string `json:"name" validate:"required"`
}

type structMapEntries[K comparable] struct {
	Items map[K]structMapEntry `json:"items" validate:""`
}

func checkStructMapKey[K comparable](t *testing.T, key K) {
	t.Helper()
	for _, valid := range []bool{true, false} {
		t.Run(fmt.Sprint(valid), func(t *testing.T) {
			entry := structMapEntry{}
			if valid {
				entry.Name = "Ada"
			}
			form := &structMapEntries[K]{Items: map[K]structMapEntry{key: entry}}
			v := Struct(form)
			assert.Eq(t, valid, v.Validate())
			if !valid {
				assert.NotEmpty(t, v.Errors.Field(fmt.Sprintf("items.%v.name", key)))
			}
			assert.Eq(t, valid, CheckErr(form) == nil)
		})
	}
}

func TestStructMapKeyCascade(t *testing.T) {
	t.Run("plain string", func(t *testing.T) { checkStructMapKey(t, "42") })
	t.Run("defined string", func(t *testing.T) { checkStructMapKey(t, structMapStringKey("42")) })
	t.Run("int", func(t *testing.T) { checkStructMapKey(t, 42) })
	t.Run("defined int", func(t *testing.T) { checkStructMapKey(t, structMapIntKey(42)) })
	t.Run("negative int", func(t *testing.T) { checkStructMapKey(t, -42) })
	t.Run("int8 minimum", func(t *testing.T) { checkStructMapKey(t, int8(math.MinInt8)) })
	t.Run("uint8 maximum", func(t *testing.T) { checkStructMapKey(t, uint8(math.MaxUint8)) })
	t.Run("int64 minimum", func(t *testing.T) { checkStructMapKey(t, int64(math.MinInt64)) })
	t.Run("uint64 maximum", func(t *testing.T) { checkStructMapKey(t, uint64(math.MaxUint64)) })
}

type structMapFilteredEntry struct {
	Name string `validate:"required" filter:"trim"`
}

type structMapDefaultEntry struct {
	Name string `validate:"required|default:Ada"`
}

func checkStructMapKeyWriteback[K comparable](t *testing.T, key K) {
	t.Helper()
	t.Run("filter", func(t *testing.T) {
		form := &struct {
			Items map[K]*structMapFilteredEntry `validate:""`
		}{Items: map[K]*structMapFilteredEntry{key: {Name: " Ada "}}}
		v := Struct(form)
		assert.True(t, v.Validate())
		assert.Eq(t, "Ada", form.Items[key].Name)
	})
	t.Run("default", func(t *testing.T) {
		form := &struct {
			Items map[K]*structMapDefaultEntry `validate:""`
		}{Items: map[K]*structMapDefaultEntry{key: {}}}
		v := Struct(form)
		assert.True(t, v.Validate())
		assert.Eq(t, "Ada", form.Items[key].Name)
	})
	t.Run("Get TryGet Set", func(t *testing.T) {
		form := &struct {
			Items map[K]*structMapEntry `validate:""`
		}{Items: map[K]*structMapEntry{key: {Name: "Ada"}}}
		d, err := FromStruct(form)
		assert.NoErr(t, err)
		path := fmt.Sprintf("Items.%v.Name", key)
		value, exist, zero := d.TryGet(path)
		assert.True(t, exist)
		assert.False(t, zero)
		assert.Eq(t, "Ada", value)
		_, err = d.Set(path, "Grace")
		assert.NoErr(t, err)
		value, exist = d.Get(path)
		assert.True(t, exist)
		assert.Eq(t, "Grace", value)
		assert.Eq(t, "Grace", form.Items[key].Name)
		// Resolve without Set's value cache, then remove the entry before writing.
		d, err = FromStruct(form)
		assert.NoErr(t, err)
		_, exist = d.Get(path)
		assert.True(t, exist)
		delete(form.Items, key)
		_, err = d.Set(path, "removed")
		assert.Error(t, err)
		// A fresh source has no cached field value after the map entry is removed.
		d, err = FromStruct(form)
		assert.NoErr(t, err)
		value, exist, zero = d.TryGet(path)
		assert.False(t, exist)
		assert.False(t, zero)
		assert.Nil(t, value)
		_, err = d.Set(path, "missing")
		assert.Error(t, err)
	})
}

func TestStructMapKeyWriteback(t *testing.T) {
	t.Run("plain string", func(t *testing.T) { checkStructMapKeyWriteback(t, "42") })
	t.Run("defined string", func(t *testing.T) { checkStructMapKeyWriteback(t, structMapStringKey("42")) })
	t.Run("int", func(t *testing.T) { checkStructMapKeyWriteback(t, 42) })
	t.Run("defined int", func(t *testing.T) { checkStructMapKeyWriteback(t, structMapIntKey(42)) })
}

func TestStructMapKeyMissingAndOverflow(t *testing.T) {
	t.Run("signed", func(t *testing.T) {
		form := &struct {
			Items map[int8]*structMapEntry `validate:""`
		}{Items: map[int8]*structMapEntry{math.MinInt8: {Name: "Ada"}}}
		d, err := FromStruct(form)
		assert.NoErr(t, err)
		for _, node := range []string{"128", "-129", "bad", "1"} {
			value, exists, zero := d.TryGet("Items." + node + ".Name")
			assert.False(t, exists)
			assert.False(t, zero)
			assert.Nil(t, value)
		}
	})
	t.Run("unsigned", func(t *testing.T) {
		form := &struct {
			Items map[uint8]*structMapEntry `validate:""`
		}{Items: map[uint8]*structMapEntry{math.MaxUint8: {Name: "Ada"}}}
		d, err := FromStruct(form)
		assert.NoErr(t, err)
		for _, node := range []string{"-1", "256", "bad", "1"} {
			value, exists, zero := d.TryGet("Items." + node + ".Name")
			assert.False(t, exists)
			assert.False(t, zero)
			assert.Nil(t, value)
		}
	})
}

// Keep string path lookups working when a map accepts string keys through an
// interface type. The path must not be converted to other dynamic key types.
type structMapInterfaceKey interface{}

func checkStructMapInterfaceKey[K comparable](t *testing.T, key K) {
	t.Helper()
	entry := &structMapEntry{Name: "Ada"}
	form := &struct {
		Items map[K]*structMapEntry
	}{Items: map[K]*structMapEntry{key: entry}}
	d, err := FromStruct(form)
	assert.NoErr(t, err)
	value, exist := d.Get("Items.alpha.Name")
	assert.True(t, exist)
	assert.Eq(t, "Ada", value)
	value, exist, zero := d.TryGet("Items.alpha.Name")
	assert.True(t, exist)
	assert.False(t, zero)
	assert.Eq(t, "Ada", value)
	_, err = d.Set("Items.alpha.Name", "Grace")
	assert.NoErr(t, err)
	assert.Eq(t, "Grace", entry.Name)
}

func TestStructMapKeyInterfaceCompatibility(t *testing.T) {
	t.Run("interface", func(t *testing.T) { checkStructMapInterfaceKey(t, any("alpha")) })
	t.Run("defined interface", func(t *testing.T) { checkStructMapInterfaceKey(t, structMapInterfaceKey("alpha")) })
	t.Run("dynamic integer remains missing", func(t *testing.T) {
		entry := &structMapEntry{Name: "Ada"}
		form := &struct {
			Items map[any]*structMapEntry
		}{Items: map[any]*structMapEntry{42: entry}}
		d, err := FromStruct(form)
		assert.NoErr(t, err)
		_, exist := d.Get("Items.42.Name")
		assert.False(t, exist)
		_, err = d.Set("Items.42.Name", "Grace")
		assert.Eq(t, ErrNoField, err)
		assert.Eq(t, "Ada", entry.Name)
	})
}
