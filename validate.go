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
	"strings"
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

// Rules definition
type Rules []*Rule

// some global vars
var (
	rulesCaches map[string]Rules
	emptyValue  = reflect.Value{}
)

/*************************************************************
 * validation rule
 *************************************************************/

// Rule definition
type Rule struct {
	// eg "create" "update"
	scene string
	// need validate fields. allow multi. eg "field1, field2"
	fields string
	// is optional, only validate on value is not empty.
	optional bool
	// default value setting
	defValue interface{}
	// error message(s)
	message  string
	messages map[string]string
	// filter map. can with args. eg. "int", "str2arr:,"
	filters map[string]string
	// validator name, allow multi validators. eg "min", "range", "required"
	validator string
	// arguments for the validator
	arguments []interface{}
	// some functions
	beforeFunc func(field string, v *Validation) bool // func (val interface{}) bool
	filterFunc func(val interface{}) (newVal interface{}, err error)
	// custom check func's reflect.Value
	checkFunc reflect.Value
	// custom check is empty.
	emptyChecker func(val interface{}) bool
}

// NewRule instance
func NewRule(fields, validator string, args ...interface{}) *Rule {
	return &Rule{
		fields: fields,
		// filters
		filters: make(map[string]string),
		// validator args
		arguments: args,
		validator: validator,
	}
}

// Setting the rule
func (r *Rule) Setting(fn func(r *Rule)) *Rule {
	fn(r)
	return r
}

// SetScene name for the rule.
func (r *Rule) SetScene(scene string) *Rule {
	r.scene = scene
	return r
}

// SetCheckFunc use custom check func.
func (r *Rule) SetCheckFunc(checkFunc interface{}) *Rule {
	r.checkFunc = checkValidatorFunc("rule.checkFunc", checkFunc)
	return r
}

// SetOptional only validate on value is not empty.
func (r *Rule) SetOptional(optional bool) *Rule {
	r.optional = optional
	return r
}

// SetMessage set error message.
// Usage:
//	v.AddRule("name", "required").SetMessage("error message")
//
func (r *Rule) SetMessage(errMsg string) *Rule {
	r.message = errMsg
	return r
}

// SetMessages set error message map.
// Usage:
//	v.AddRule("name,email", "required").SetMessages(MS{
// 		"name": "error message 1",
// 		"email": "error message 2",
// 	})
func (r *Rule) SetMessages(msgMap MS) *Rule {
	r.messages = msgMap
	return r
}

// UseFilters add filter(s)
func (r *Rule) UseFilters(filters ...string) *Rule {
	for _, filterN := range filters {
		pos := strings.IndexRune(filterN, ':')

		// has args
		if pos > 0 {
			name := filterN[:pos]
			r.filters[name] = filterN[pos+1:]
		} else {
			r.filters[filterN] = ""
		}
	}

	return r
}

// Fields names list
func (r *Rule) Fields() []string {
	return stringSplit(r.fields, ",")
}

// Apply rule for the rule fields
func (r *Rule) Apply(v *Validation) (stop bool) {
	// scene name is not match.
	if r.scene != "" && r.scene != v.scene {
		return false
	}

	// validate field value
	for _, field := range r.Fields() {
		if v.isNoNeedToCheck(field) {
			continue
		}

		val, has := v.Get(field)   // get field value.
		if !has && v.StopOnError { // no field AND stop on error
			return true
		}

		// apply filters func
		val, err := applyFilters(val, r.filters, v)
		if err != nil { // has error
			v.AddError(filterError, err.Error())
			return true
		} else { // save filtered value.
			v.filteredData[field] = val
		}

		// only one validator
		if !strings.ContainsRune(r.validator, '|') {
			r.Validate(field, r.validator, val, v)
		} else { // has multi validators
			vs := stringSplit(r.validator, "|")
			for _, validator := range vs {
				// stop on error
				if r.Validate(field, validator, val, v) && v.StopOnError {
					return true
				}
			}
		}

		// stop on error
		if v.shouldStop() {
			return true
		}

		// save validated value.
		v.safeData[field] = val
	}

	return false
}

func (r *Rule) errorMessage(field string) (msg string, ok bool) {
	if r.messages != nil {
		msg, ok = r.messages[field]
		if ok {
			return
		}
	}

	if r.message != "" {
		return r.message, true
	}

	return
}

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
	if r.checkFunc.IsValid() {
		ok = callValidatorValue(validator, r.checkFunc, val, r.arguments)
	} else if fv, has := v.ValidatorValue(validator); has { // find validator
		ok = callValidatorValue(validator, fv, val, r.arguments)
	} else {
		panicf("the validator '%s' is not exists", validator)
	}

	// build and collect error message
	if !ok {
		errMsg, has := r.errorMessage(field)
		if !has {
			errMsg = v.trans.Message(validator, field, r.arguments...)
		}

		v.AddError(field, errMsg)
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
