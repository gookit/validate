package validate

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

type ExtraSt struct {
	Email string
}

type TestSt struct {
	Name    string `json:"name" validate:"required,minLength(2)"`
	Age     int    `json:"age" validate:"range(23, 100)"`
	ExtraSt        // is Anonymous field
	pwd     string
}

func TestSome(t *testing.T) {
	s := "str"
	rv := reflect.ValueOf(s)
	rt := rv.Type()
	fmt.Println(rt.Kind(), rv.String())

	m := map[string]string{"a": "v"}
	rv = reflect.ValueOf(m)
	rt = rv.Type()
	fmt.Println(rt.Key(), rv.Len())
	fmt.Printf("%+v\n", rv.MapKeys())

	st := new(TestSt)
	rv = reflect.ValueOf(st)
	rt = reflect.TypeOf(st)

	// 如果当前是指针，需要转换为值
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
		rv = rv.Elem()
	}

	fmt.Println(rt.Kind(), rt.PkgPath())
	fmt.Printf("%+v\n", rt.Field(0))
	fmt.Printf("%s\n", rt.Field(0).Tag.Get("validate"))
	fmt.Printf("%v\n", rt.Field(3).Name)
	fmt.Println(rv.Kind(), rv.Field(0).String())
}

func TestIsInt(t *testing.T) {
	is := assert.New(t)

	// type check
	is.True(IsInt(2))
	is.True(IsInt(int8(2)))
	is.True(IsInt(int16(2)))
	is.True(IsInt(int32(2)))
	is.True(IsInt(int64(2)))
	is.False(IsInt(nil))
	is.False(IsInt("str"))
	is.False(IsInt([]int{}))
	is.False(IsInt([]int{2}))
	is.False(IsInt(map[string]int{"key": 2}))

	// with min and max value
	is.True(IsInt(5, 5))
	is.True(IsInt(5, 4))
	is.True(IsInt(5, 4, 6))
	is.False(IsInt(nil, 4, 6))
	is.False(IsInt("str", 4, 6))
}

func TestMin(t *testing.T) {
	is := assert.New(t)

	// ok
	is.True(Min(3, 2))
	is.True(Min(3, 3))
	is.True(Min(int64(3), 3))

	// fail
	is.False(Min(nil, 3))
	is.False(Min("str", 3))
	is.False(Min(3, 4))
	is.False(Min(int64(3), 4))
}

func TestMax(t *testing.T) {
	is := assert.New(t)

	// ok
	is.True(Max(3, 4))
	is.True(Max(3, 3))
	is.True(Max(int64(3), 3))

	// fail
	is.False(Max(nil, 3))
	is.False(Max("str", 3))
	is.False(Max(3, 2))
	is.False(Max(int64(3), 2))
}

func TestIsString(t *testing.T) {
	is := assert.New(t)

	is.True(IsString("str"))
	is.False(IsString(nil))
	is.False(IsString(2))

	is.True(IsString("str", 3))
	is.True(IsString("str", 3, 5))
	is.False(IsString("str", 4))
	is.False(IsString("str", 1, 2))
}

func TestIsAlpha(t *testing.T) {
	var val interface{}
	val = "val"

	fmt.Println(val, reflect.TypeOf(val).Kind())
}
