package validate

import (
	"reflect"
	"testing"

	"github.com/gookit/goutil/dump"
	"github.com/gookit/goutil/testutil/assert"
)

func TestValueLen(t *testing.T) {
	is := assert.New(t)
	tests := []any{
		"abc",
		123,
		int8(123), int16(123), int32(123), int64(123),
		uint8(123), uint16(123), uint32(123), uint64(123),
		float32(123), float64(123),
		[]int{1, 2, 3}, []string{"a", "b", "c"},
		map[string]string{"k0": "v0", "k1": "v1", "k2": "v2"},
	}

	for _, sample := range tests {
		is.Equal(3, ValueLen(reflect.ValueOf(sample)))
	}

	ptrArr := &[]string{"a", "b"}
	is.Equal(2, ValueLen(reflect.ValueOf(ptrArr)))
	is.Equal(4, ValueLen(reflect.ValueOf("ab你好")))

	is.Equal(-1, ValueLen(reflect.ValueOf(nil)))
}

func TestFlatSlice(t *testing.T) {
	sl := []any{
		[]string{"a", "b"},
	}
	fsl := flatSlice(reflect.ValueOf(sl), 1)
	// dump.P(fsl.Interface())
	assert.Equal(t, 2, fsl.Len())
	assert.Equal(t, 2, fsl.Cap())

	// make slice len=2, cap=3
	sub1 := make([]string, 0, 3)
	sub1 = append(sub1, "a", "b")

	sl = []any{
		sub1,
	}
	fsl = flatSlice(reflect.ValueOf(sl), 1)
	dump.P(fsl.Interface())
	assert.Equal(t, 2, fsl.Len())
	assert.Equal(t, 3, fsl.Cap())

	sl = []any{
		[]string{"a", "b"},
		sub1,
	}
	fsl = flatSlice(reflect.ValueOf(sl), 1)
	// dump.P(fsl.Interface())
	assert.Equal(t, 4, fsl.Len())
	assert.Equal(t, 5, fsl.Cap())

	// 3 level
	sl = []any{
		[]any{
			[]string{"a", "b"},
		},
	}

	fsl = flatSlice(reflect.ValueOf(sl), 2)
	dump.P(fsl.Interface())
	assert.Equal(t, 2, fsl.Len())
	assert.Equal(t, 2, fsl.Cap())
}

func TestCallByValue(t *testing.T) {
	is := assert.New(t)
	is.Panics(func() {
		CallByValue(reflect.ValueOf("invalid"))
	})
	is.Panics(func() {
		CallByValue(reflect.ValueOf(IsJSON), "age0", "age1")
	})

	rs := CallByValue(reflect.ValueOf(IsNumeric), "123")
	is.Len(rs, 1)
	is.Equal(true, rs[0].Interface())
}

func TestCallByValue_nil_arg(t *testing.T) {
	fn1 := func(in any) any {
		_, ok := in.(NilObject)
		assert.True(t, IsNilObj(in))
		dump.P(in, ok)
		return in
	}

	// runtime error: invalid memory address or nil pointer dereference
	// typ := reflect.TypeOf(any(nil))
	// typ.Kind()

	nilV := 2

	dump.P(
		reflect.ValueOf(nilV).Kind().String(),
		// reflect.New(reflect.Interface).Kind(),
	)

	rs := CallByValue(reflect.ValueOf(fn1), nil)
	dump.P(rs[0].CanInterface(), rs[0].Interface())
}
