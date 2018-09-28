// Package validate is a generic go data validate library.
//
// Source code and other details for the project are available at GitHub:
//
// 	https://github.com/gookit/validate
//
package validate

import (
	"fmt"
	"github.com/gookit/filter"
	"reflect"
)

type sourceType uint8

const (
	// from user setting, unmarshal JSON
	sourceMap sourceType = iota + 1
	// from URL.Values, PostForm. contains Files data
	sourceForm
	// from user setting
	sourceStruct
)

var emptyValue = reflect.Value{}

// Validate the field by validator name
func (r *Rule) Validate(field, validator string, val interface{}, v *Validation) (ok bool) {
	// "-" OR "safe" mark field value always is safe.
	if validator == "-" || validator == "safe" {
		return true
	}

	// beforeFunc return false, skip validate
	if r.beforeFunc != nil && !r.beforeFunc(field, v) {
		return false
	}

	// call custom validator
	if r.checkFunc != nil {
		ok = callValidatorFunc(validator, r.checkFunc, val, r.arguments)
	} else if fv, has := v.ValidatorValue(validator); has { // find validator
		ok = callValidatorValue(validator, fv, val, r.arguments)
	} else {
		panicf("the validator '%s' is not exists", validator)
	}

	// build and collect error message
	if !ok {
		v.AddError(field, v.trans.Message(validator, field, r.arguments...))
	}

	return
}

func callValidatorFunc(name string, fn, val interface{}, args []interface{}) bool {
	fv := reflect.ValueOf(fn)
	if fv.Kind() != reflect.Func {
		panicf("validator '%s' func must be an func type", name)
	}

	return callValidatorValue(name, fv, val, args)
}

func callValidatorValue(name string, fv reflect.Value, val interface{}, args []interface{}) bool {
	ft := fv.Type()
	if ft.NumOut() != 1 || ft.Out(0).Kind() != reflect.Bool {
		panicf("the validator '%s' func must be return a bool value.", name)
	}

	fnArgNum := ft.NumIn() // arg num for the func

	// only one param in the validator func.
	if fnArgNum == 1 {
		vs := fv.Call([]reflect.Value{reflect.ValueOf(val)})
		return vs[0].Bool()
	}

	argNum := len(args) + 1
	notEnough := argNum < fnArgNum

	// last arg is like "... interface{}"
	if ft.IsVariadic() {
		notEnough = argNum+1 < fnArgNum
	}

	if notEnough {
		panicf("not enough parameters for validator '%s'!", name)
	}

	newArgs := make([]interface{}, argNum)
	newArgs[0] = val
	copy(newArgs[1:], args)

	// build params for the validator func.
	argIn := make([]reflect.Value, argNum)
	// typeIn := make([]reflect.Type, fnArgNum)
	for i := 0; i < argNum; i++ {
		av := reflect.ValueOf(newArgs[i])
		wantTyp := ft.In(i).Kind()
		updateArg := false

		// compare func param type and input param type.
		if wantTyp == av.Kind() {
			argIn[i] = av
		} else if av.Type().ConvertibleTo(ft.In(i)) { // need convert type.
			updateArg = true
			argIn[i] = av.Convert(ft.In(i))
		} else if nv, ok := convertValueType(av, wantTyp); ok { // manual converted
			argIn[i] = nv
			updateArg = true
		} else { // cannot converted
			return false
		}

		// update rule.arguments[i] value
		if updateArg && i != 0 {
			args[i-1] = argIn[i].Interface()
		}
	}

	// fmt.Printf("%#v %v\n", val, argIn[0].String())

	// f.CallSlice()与Call() 不一样的是，CallSlice参数的最后一个会被展开
	// vs := fv.Call(argIn)
	return fv.Call(argIn)[0].Bool()
}

func convertValueType(src reflect.Value, dstType reflect.Kind) (nVal reflect.Value, ok bool) {
	switch src.Kind() {
	case reflect.String:
		srcVal := src.String()
		switch dstType {
		case reflect.Int:
			return convertResult(filter.Int(srcVal))
		case reflect.Int64:
			return convertResult(filter.Int64(srcVal))
		}
	case reflect.Int:
		switch dstType {
		case reflect.Int64:
			return convertResult(int64(src.Int()), nil)
		case reflect.String:
			return convertResult(fmt.Sprint(src.Int()), nil)
		}
	}

	return
}

func convertResult(val interface{}, err error) (reflect.Value, bool) {
	if err != nil {
		return emptyValue, false
	}

	return reflect.ValueOf(val), true
}
