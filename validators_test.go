package validate

import (
	"fmt"
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

func TestMin(t *testing.T) {
	f := reflect.ValueOf(Min)
	vs := CallByValue(f, int64(4), int64(3))
	fmt.Println(vs[0].Bool(), f.Type().PkgPath())

	f = reflect.ValueOf(stringSplit)
	vs = CallByValue(f, "a,b,c", ",")
	fmt.Printf("%#v\n", vs[0])

	f = reflect.ValueOf(ToArray)
	vs = CallByValue(f, "a,b,c", ",")
	// .Interface().([]string)
	fmt.Printf("%#v\n", vs[0].Interface().([]string))
}

func TestIsAlpha(t *testing.T) {
	var val interface{}
	val = "val"

	fmt.Println(val, reflect.TypeOf(val).Kind())
}
