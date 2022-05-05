package validate

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/gookit/filter"
	"github.com/gookit/goutil/mathutil"
	"github.com/gookit/goutil/strutil"
)

// NilObject represent nil value for calling functions and should be reflected at custom filters as nil variable.
type NilObject struct{}

// CallByValue call func by reflect.Value
func CallByValue(fv reflect.Value, args ...interface{}) []reflect.Value {
	if fv.Kind() != reflect.Func {
		panicf("parameter must be an func type")
	}

	in := make([]reflect.Value, len(args))
	for k, v := range args {
		// NOTICE: reflect.Call emit panic if kind is Invalid
		if in[k] = reflect.ValueOf(v); in[k].Kind() == reflect.Invalid {
			in[k] = reflect.ValueOf(NilObject{})
		}
	}

	// NOTICE: CallSlice()与Call() 不一样的是，参数的最后一个会被展开
	// f.CallSlice()
	return fv.Call(in)
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

// convert []interface{} to string TODO use arrutil.ToString()
func sliceToString(arr []interface{}) string {
	var b strings.Builder
	b.WriteByte('[')

	for i, v := range arr {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strutil.MustString(v))
	}

	b.WriteByte(']')
	return b.String()
}

func strings2Args(strings []string) []interface{} {
	args := make([]interface{}, len(strings))
	for i, s := range strings {
		args[i] = s
	}
	return args
}

func args2strings(args []interface{}) []string {
	strSlice := make([]string, len(args))
	for i, a := range args {
		strSlice[i] = strutil.MustString(a)
	}
	return strSlice
}

func buildArgs(val interface{}, args []interface{}) []interface{} {
	newArgs := make([]interface{}, len(args)+1)
	newArgs[0] = val
	// as[1:] = args // error
	copy(newArgs[1:], args)

	return newArgs
}

// ValueIsEmpty check
func ValueIsEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Invalid:
		return true
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
	case reflect.Float32, reflect.Float64:
		return len(fmt.Sprint(v.Interface()))
	}

	// cannot get length
	return -1
}

var (
	ErrConvertFail = errors.New("convert value is failure")
)

func valueToInt64(v interface{}, strict bool) (i64 int64, err error) {
	switch tVal := v.(type) {
	case string:
		if strict {
			return 0, ErrConvertFail
		}
		i64, err = strconv.ParseInt(filter.Trim(tVal), 10, 0)
	case int:
		i64 = int64(tVal)
	case int8:
		i64 = int64(tVal)
	case int16:
		i64 = int64(tVal)
	case int32:
		i64 = int64(tVal)
	case int64:
		i64 = tVal
	case uint:
		i64 = int64(tVal)
	case uint8:
		i64 = int64(tVal)
	case uint16:
		i64 = int64(tVal)
	case uint32:
		i64 = int64(tVal)
	case uint64:
		i64 = int64(tVal)
	case float32:
		if strict {
			return 0, ErrConvertFail
		}
		i64 = int64(tVal)
	case float64:
		if strict {
			return 0, ErrConvertFail
		}
		i64 = int64(tVal)
	default:
		err = ErrConvertFail
	}
	return
}

// CalcLength for input value
func CalcLength(val interface{}) int {
	if val == nil {
		return -1
	}

	// string length
	if str, ok := val.(string); ok {
		// return len(str)
		// fix: issues#39
		return len([]rune(str))
	}

	return ValueLen(reflect.ValueOf(val))
}

// value compare. use for compare int, string.
func valueCompare(srcVal, dstVal interface{}, op string) (ok bool) {
	var err error
	var srcInt, dstInt int64

	// string: compare length
	if str, ok := srcVal.(string); ok {
		dst, ok := dstVal.(string)
		if !ok {
			return false
		}

		srcInt = int64(len(str))
		dstInt = int64(len(dst))
	} else { // as int: compare size
		srcInt, err = mathutil.ToInt64(srcVal)
		if err != nil {
			return false
		}

		dstInt, err = mathutil.ToInt64(dstVal)
		if err != nil {
			return false
		}
	}

	return compareInt64(srcInt, dstInt, op)
}

// compare int float value. returns `srcVal op(lt,lte,gt,gte) dstVal`?
func compareIntFloat(srcVal, dstVal interface{}, op string) (ok bool) {
	if srcVal == nil || dstVal == nil {
		return false
	}

	if srcFlt, ok := srcVal.(float64); ok {
		dstFlt, err := mathutil.ToFloat(dstVal)
		if err != nil {
			return false
		}
		return compareFloat64(srcFlt, dstFlt, op)
	}

	if srcFlt, ok := srcVal.(float32); ok {
		dstFlt, err := mathutil.ToFloat(dstVal)
		if err != nil {
			return false
		}
		return compareFloat64(float64(srcFlt), dstFlt, op)
	}

	// as int64
	srcInt, err := mathutil.Int64(srcVal)
	if err != nil {
		return false
	}

	dstInt, err := mathutil.Int64(dstVal)
	if err != nil {
		return false
	}

	return compareInt64(srcInt, dstInt, op)
}

// compare int64, returns the srcI64 op(lt,lte,gt,gte) dstI64?
func compareInt64(srcI64, dstI64 int64, op string) (ok bool) {
	switch op {
	case "lt":
		ok = srcI64 < dstI64
	case "lte":
		ok = srcI64 <= dstI64
	case "gt":
		ok = srcI64 > dstI64
	case "gte":
		ok = srcI64 >= dstI64
	}
	return
}

func compareFloat64(srcI64, dstI64 float64, op string) (ok bool) {
	switch op {
	case "lt":
		ok = srcI64 < dstI64
	case "lte":
		ok = srcI64 <= dstI64
	case "gt":
		ok = srcI64 > dstI64
	case "gte":
		ok = srcI64 >= dstI64
	}
	return
}

// func nameOfFunc(fv reflect.Value) string {
// 	return runtime.FuncForPC(fv.Pointer()).Name()
// }

func parseArgString(argStr string) (ss []string) {
	if argStr == "" { // no arg
		return
	}

	if len(argStr) == 1 { // one char
		return []string{argStr}
	}

	return stringSplit(argStr, ",")
}

func toInt64Slice(enum interface{}) (ret []int64, ok bool) {
	rv := reflect.ValueOf(enum)
	if rv.Kind() != reflect.Slice {
		return
	}

	for i := 0; i < rv.Len(); i++ {
		i64, err := mathutil.ToInt64(rv.Index(i).Interface())
		if err != nil {
			return []int64{}, false
		}

		ret = append(ret, i64)
	}

	ok = true
	return
}

func getVariadicKind(typString string) reflect.Kind {
	switch typString {
	case "[]int":
		return reflect.Int
	case "[]int8":
		return reflect.Int8
	case "[]int16":
		return reflect.Int16
	case "[]int32":
		return reflect.Int32
	case "[]int64":
		return reflect.Int64
	case "[]uint":
		return reflect.Uint
	case "[]uint8":
		return reflect.Uint8
	case "[]uint16":
		return reflect.Uint16
	case "[]uint32":
		return reflect.Uint32
	case "[]uint64":
		return reflect.Uint64
	case "[]string":
		return reflect.String
	case "[]interface {}": // args ...interface{}
		return reflect.Interface
	}
	return reflect.Invalid
}

func convertType(srcVal interface{}, srcKind kind, dstType reflect.Kind) (interface{}, error) {
	switch srcKind {
	case stringKind:
		switch dstType {
		case reflect.Int:
			return mathutil.Int(srcVal)
		case reflect.Int64:
			return mathutil.Int64(srcVal)
		case reflect.Bool:
			return strutil.Bool(srcVal.(string))
		}
	case intKind, uintKind:
		i64 := mathutil.MustInt64(srcVal)
		switch dstType {
		case reflect.Int64:
			return i64, nil
		case reflect.String:
			// fmt is slow : return fmt.Sprint(i64), nil
			return strutil.ToString(srcVal)
		}
	default:
		switch dstType {
		case reflect.String:
			return strutil.ToString(srcVal)
		}
	}
	return nil, ErrConvertFail
}

func panicf(format string, args ...interface{}) {
	panic("validate: " + fmt.Sprintf(format, args...))
}

// From package "text/template" -> text/template/funcs.go
var (
	errorType = reflect.TypeOf((*error)(nil)).Elem()
	// fmtStringerType  = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()
	// reflectValueType = reflect.TypeOf((*reflect.Value)(nil)).Elem()
)

func checkValidatorFunc(name string, fn interface{}) reflect.Value {
	if !goodName(name) {
		panicf("validate name %s is not a valid identifier", name)
	}

	fv := reflect.ValueOf(fn)
	if fn == nil || fv.Kind() != reflect.Func { // is nil or not is func
		panicf("validator '%s'. 2th parameter is invalid, it must be an func", name)
	}

	ft := fv.Type()
	if ft.NumIn() == 0 {
		panicf("validator '%s' func at least one parameter position", name)
	}

	if ft.NumOut() != 1 || ft.Out(0).Kind() != reflect.Bool {
		panicf("validator '%s' func must be return a bool value", name)
	}

	return fv
}

func checkFilterFunc(name string, fn interface{}) reflect.Value {
	if !goodName(name) {
		panic(fmt.Errorf("filter name %s is not a valid identifier", name))
	}

	fv := reflect.ValueOf(fn)
	if fn == nil || fv.Kind() != reflect.Func { // is nil or not is func
		panicf("filter '%s'. 2th parameter is invalid, it must be an func", name)
	}

	ft := fv.Type()
	if ft.NumIn() == 0 {
		panicf("filter '%s' func at least one parameter position", name)
	}

	if !goodFunc(ft) {
		panicf("can't install method/function %q with %d results", name, ft.NumOut())
	}

	return fv
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

// ---- From package "text/template" -> text/template/exec.go
// indirect returns the item at the end of indirection, and a bool to indicate if it's nil.
// func indirect(v reflect.Value) (rv reflect.Value, isNil bool) {
// 	for ; v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface; v = v.Elem() {
// 		if v.IsNil() {
// 			return v, true
// 		}
// 	}
// 	return v, false
// }

// indirectInterface returns the concrete value in an interface value,
// or else the zero reflect.Value.
// That is, if v represents the interface value x, the result is the same as reflect.ValueOf(x):
// the fact that x was an interface value is forgotten.
func indirectInterface(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Interface {
		return v
	}

	if v.IsNil() {
		return emptyValue
	}

	return v.Elem()
}

/*************************************************************
 * Comparison:
 * From package "text/template" -> text/template/funcs.go
 *************************************************************/

// TODO: Perhaps allow comparison between signed and unsigned integers.

var (
	errBadComparisonType = errors.New("invalid type for operation")
	// errBadComparison     = errors.New("incompatible types for comparison")
	// errNoComparison      = errors.New("missing argument for comparison")
)

type kind int

const (
	invalidKind kind = iota
	boolKind
	complexKind
	intKind
	floatKind
	stringKind
	uintKind
)

func basicKind(v reflect.Value) (kind, error) {
	switch v.Kind() {
	case reflect.Bool:
		return boolKind, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return intKind, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return uintKind, nil
	case reflect.Float32, reflect.Float64:
		return floatKind, nil
	case reflect.Complex64, reflect.Complex128:
		return complexKind, nil
	case reflect.String:
		return stringKind, nil
	}

	// like: slice, array, map ...
	return invalidKind, errBadComparisonType
}

// eq evaluates the comparison a == b
func eq(arg1 reflect.Value, arg2 reflect.Value) (bool, error) {
	v1 := indirectInterface(arg1)
	k1, err := basicKind(v1)
	if err != nil {
		return false, err
	}

	v2 := indirectInterface(arg2)
	k2, err := basicKind(v2)
	if err != nil {
		return false, err
	}

	truth := false
	if k1 != k2 {
		// Special case: Can compare integer values regardless of type's sign.
		switch {
		case k1 == intKind && k2 == uintKind:
			truth = v1.Int() >= 0 && uint64(v1.Int()) == v2.Uint()
		case k1 == uintKind && k2 == intKind:
			truth = v2.Int() >= 0 && v1.Uint() == uint64(v2.Int())
			// default:
			// 	 return false, errBadComparison
		}
		return truth, nil
	}

	switch k1 {
	case boolKind:
		truth = v1.Bool() == v2.Bool()
	case complexKind:
		truth = v1.Complex() == v2.Complex()
	case floatKind:
		truth = v1.Float() == v2.Float()
	case intKind:
		truth = v1.Int() == v2.Int()
	case stringKind:
		truth = v1.String() == v2.String()
	case uintKind:
		truth = v1.Uint() == v2.Uint()
		// default:
		// 	panic("invalid kind")
	}

	return truth, nil
}

// from package: github.com/stretchr/testify/assert/assertions.go
func includeElement(list, element interface{}) (ok, found bool) {
	listValue := reflect.ValueOf(list)
	elementValue := reflect.ValueOf(element)
	listKind := listValue.Type().Kind()

	// string contains check
	if listKind == reflect.String {
		return true, strings.Contains(listValue.String(), elementValue.String())
	}

	defer func() {
		if e := recover(); e != nil {
			ok = false // call Value.Len() panic.
			found = false
		}
	}()

	if listKind == reflect.Map {
		mapKeys := listValue.MapKeys()
		for i := 0; i < len(mapKeys); i++ {
			if IsEqual(mapKeys[i].Interface(), element) {
				return true, true
			}
		}
		return true, false
	}

	for i := 0; i < listValue.Len(); i++ {
		if IsEqual(listValue.Index(i).Interface(), element) {
			return true, true
		}
	}

	return true, false
}

/*************************************************************
 * Reflection:
 * From package(go 1.13) "reflect" -> reflect/value.go
 *************************************************************/

// IsZero reports whether v is the zero value for its type.
// It panics if the argument is invalid.
// NOTICE: this built-in method in reflect/value.go since go 1.13
func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return math.Float64bits(v.Float()) == 0
	case reflect.Complex64, reflect.Complex128:
		c := v.Complex()
		return math.Float64bits(real(c)) == 0 && math.Float64bits(imag(c)) == 0
	case reflect.Array:
		for i := 0; i < v.Len(); i++ {
			// if !v.Index(i).IsZero() {
			if !IsZero(v.Index(i)) {
				return false
			}
		}
		return true
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
		return v.IsNil()
	case reflect.String:
		return v.Len() == 0
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			// if !v.Index(i).IsZero() {
			if !IsZero(v.Field(i)) {
				return false
			}
		}
		return true
	default:
		// This should never happen, but will act as a safeguard for
		// later, as a default value doesn't make sense here.
		panic(&reflect.ValueError{Method: "cannot check reflect.Value.IsZero", Kind: v.Kind()})
	}
}

// Remove type multiple pointer
func removeTypePtr(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

// Remove value multiple pointer
func removeValuePtr(t reflect.Value) reflect.Value {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}
