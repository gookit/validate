package validate

import (
	"fmt"
	"reflect"
)

// CallByValue call func by reflect.Value
func CallByValue(fv reflect.Value, args ...interface{}) []reflect.Value {
	if fv.Kind() != reflect.Func {
		panic("parameter must be an func type")
	}

	argNum := len(args)
	if argNum < fv.Type().NumIn() {
		fmt.Println("the number of input params not match!")
	}

	in := make([]reflect.Value, argNum)
	for k, v := range args {
		in[k] = reflect.ValueOf(v)
	}

	// CallSlice()与Call() 不一样的是，参数的最后一个会被展开
	// f.CallSlice()
	return fv.Call(in)
}

// Call call func by reflection
func Call(fn interface{}, args ...interface{}) []reflect.Value {
	return CallByValue(reflect.ValueOf(fn), args...)
}
