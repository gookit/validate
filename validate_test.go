package validate

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUtil_Func_valueToInt64(t *testing.T)  {
	noErrTests := []struct {
		val interface{}
		strict bool
		want int64
	}{
		{" 12", false, 12},
		{float32(12.23), false, 12},
		{12.23, false, 12},
	}

	for _, item := range noErrTests {
		i64, err := valueToInt64(item.val, item.strict)
		assert.NoError(t, err)
		assert.Equal(t, item.want, i64)
	}
}

func TestUtil_Func_getVariadicKind(t *testing.T)  {
	noErrTests := []struct {
		val interface{}
		want reflect.Kind
	}{
		{"invalid",  reflect.Invalid},
		{[]int{1, 2},  reflect.Int},
		{[]int8{1, 2},  reflect.Int8},
		{[]int16{1, 2},  reflect.Int16},
		{[]int32{1, 2},  reflect.Int32},
		{[]int64{1, 2},  reflect.Int64},
		{[]uint{1, 2},  reflect.Uint},
		{[]uint8{1, 2},  reflect.Uint8},
		{[]uint16{1, 2},  reflect.Uint16},
		{[]uint32{1, 2},  reflect.Uint32},
		{[]uint64{1, 2},  reflect.Uint64},
	}

	for _, item := range noErrTests {
		vt := reflect.TypeOf(item.val)
		eleType := getVariadicKind(vt.String())
		assert.Equal(t, item.want, eleType)
	}
}

func TestMS_String(t *testing.T) {
	ms := MS{}

	assert.Equal(t, "", ms.One())
	assert.Equal(t, "", ms.String())

	ms["key"] = "val"
	assert.Equal(t, "val", ms.One())
	assert.Equal(t, " key: val", ms.String())
}

func TestOption(t *testing.T) {
	opt := Option()

	assert.Equal(t, "json", opt.FieldTag)
	assert.Equal(t, "validate", opt.ValidateTag)

	Config(func(opt *GlobalOption) {
		opt.ValidateTag = "valid"
	})

	opt = Option()
	assert.Equal(t, "valid", opt.ValidateTag)

	ResetOption()
}
