package validate

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
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

func stringSplit(str, sep string) (ss []string) {
	str = strings.TrimSpace(str)
	if str == "" {
		return
	}

	for _, val := range strings.Split(str, sep) {
		if val = strings.TrimSpace(val); val != "" {
			ss = append(ss, val)
		}
	}

	return
}

func strings2Args(strings []string) []interface{} {
	args := make([]interface{}, len(strings))
	for i, s := range strings {
		args[i] = s
	}

	return args
}

func buildArgs(val interface{}, args []interface{}) []interface{} {
	newArgs := make([]interface{}, len(args)+1)
	newArgs[0] = val
	// as[1:] = args // error
	copy(newArgs[1:], args)

	return newArgs
}

// upperFirst upper first char
func upperFirst(s string) string {
	if len(s) == 0 {
		return s
	}

	f := s[0]

	if f >= 'a' && f <= 'z' {
		return strings.ToUpper(string(f)) + string(s[1:])
	}

	return s
}

// ValueIsEmpty check
func ValueIsEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String, reflect.Array:
		return v.Len() == 0
	case reflect.Map, reflect.Slice:
		return v.Len() == 0 || v.IsNil()
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}

	return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
}

// ValueInt64 get int64 value
func ValueInt64(v reflect.Value) (int64, bool) {
	k := v.Kind()
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		return int64(v.Float()), true
	}

	// cannot get int value
	return 0, false
}

// ValueLen get value length
func ValueLen(v reflect.Value) int {
	k := v.Kind()

	// (u)int use width.
	switch k {
	case reflect.Map, reflect.Array, reflect.Chan, reflect.Slice, reflect.String:
		return v.Len()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return len(fmt.Sprint(v.Uint()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return len(fmt.Sprint(v.Int()))
	}

	// cannot get length
	return -1
}

// ValueLenOrInt calc
func ValueLenOrInt(v reflect.Value) int64 {
	k := v.Kind()
	switch k {
	case reflect.Map, reflect.Array, reflect.Chan, reflect.Slice, reflect.String: // return len
		return int64(v.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return int64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return int64(v.Float())
	}

	return 0
}

// CalcLength for input value
func CalcLength(val interface{}) int {
	if val == nil {
		return -1
	}

	if rv, ok := val.(reflect.Value); ok {
		return ValueLen(rv)
	}

	return ValueLen(reflect.ValueOf(val))
}

// IntVal of the val
func IntVal(val interface{}) (intVal int64, ok bool) {
	switch tv := val.(type) {
	case int:
		ok = true
		intVal = int64(tv)
	case int64:
		ok = true
		intVal = tv
	case reflect.Value:
		intVal, ok = ValueInt64(tv)
	default:
		intVal, ok = ValueInt64(reflect.ValueOf(val))
	}

	return
}

func int64compare(intVal, dstVal int64, op string) bool {
	switch op {
	case "eq":
		return intVal == dstVal
	case "ne":
		return intVal != dstVal
	case "lt":
		return intVal < dstVal
	case "lte":
		return intVal <= dstVal
	case "gt":
		return intVal > dstVal
	case "gte":
		return intVal >= dstVal
	}

	return false
}

func toInt64Slice(enum interface{}) (ret []int64, ok bool) {
	ok = true
	switch td := enum.(type) {
	case []int:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []int8:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []int16:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []int32:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []int64:
		ret = td
	case []uint:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []uint8:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []uint16:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []uint32:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []uint64:
		for _, val := range td {
			ret = append(ret, int64(val))
		}
	case []string: // try convert string to int
		for _, val := range td {
			i64, err := strconv.ParseInt(val, 10, 0)
			if err != nil {
				ret = []int64{} // reset
				break
			}

			ret = append(ret, i64)
		}
	default:
		ok = false
	}

	return
}

func panicf(format string, args ...interface{}) {
	panic("validate: " + fmt.Sprintf(format, args...))
}

// From package "text/template" -> text/template/funcs.go
var (
	errorType        = reflect.TypeOf((*error)(nil)).Elem()
	// fmtStringerType  = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	// reflectValueType = reflect.TypeOf((*reflect.Value)(nil)).Elem()
)

// addValueFuncs adds to values the functions in funcs, converting them to reflect.Values.
func addValueFuncs(out map[string]reflect.Value, in M) {
	for name, fn := range in {
		if !goodName(name) {
			panic(fmt.Errorf("function name %s is not a valid identifier", name))
		}
		v := reflect.ValueOf(fn)
		if v.Kind() != reflect.Func {
			panic("value for " + name + " not a function")
		}
		if !goodFunc(v.Type()) {
			panic(fmt.Errorf("can't install method/function %q with %d results", name, v.Type().NumOut()))
		}
		out[name] = v
	}
}

// goodFunc reports whether the function or method has the right result signature.
func goodFunc(typ reflect.Type) bool {
	// We allow functions with 1 result or 2 results where the second is an error.
	switch {
	case typ.NumOut() == 1:
		return true
	case typ.NumOut() == 2 && typ.Out(1) == errorType:
		return true
	}
	return false
}

// goodName reports whether the function name is a valid identifier.
func goodName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		switch {
		case r == '_':
		case i == 0 && !unicode.IsLetter(r):
			return false
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			return false
		}
	}
	return true
}