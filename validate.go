// Package validate is a generic go data validate library.
package validate

import (
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

// Validate the field by validator name
func (r *Rule) Validate(field, validator string, v *Validation) (ok bool) {
	// beforeFunc return false, skip validate
	if r.beforeFunc != nil && !r.beforeFunc(field, v) {
		return false
	}

	// get field value.
	val, has := v.Get(field)
	if !has {
		return false
	}

	// call custom validator
	if r.checkFunc != nil {
		ok = callValidatorFunc(validator, r.checkFunc, val, r.arguments)
	} else {
		fv, has := v.ValidatorValue(validator)
		if !has {
			panicf("the validator '%s' is not exists", validator)
		}

		ok = callValidatorValue(validator, fv, val, r.arguments)
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
	if argNum < fnArgNum {
		panicf("not enough parameters for validator '%s'!", name)
	}

	newArgs := make([]interface{}, argNum)
	newArgs[0] = val
	copy(newArgs[1:], args)

	// build params for the validator func.
	argIn := make([]reflect.Value, fnArgNum)
	// typeIn := make([]reflect.Type, fnArgNum)
	for i := 0; i < fnArgNum; i++ {
		av := reflect.ValueOf(newArgs[i])

		// compare func param type and input param type.
		if ft.In(i).Kind() == av.Kind() {
			argIn[i] = av
		} else if av.Type().ConvertibleTo(ft.In(i)) { // need convert type.
			argIn[i] = av.Convert(ft.In(i))
		} else { // cannot converted
			return false
		}
	}

	// f.CallSlice()与Call() 不一样的是，参数的最后一个会被展开
	vs := fv.Call(argIn)
	return vs[0].Bool()
}
