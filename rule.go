package validate

import (
	"fmt"
	"reflect"
	"strings"
)

// Rules definition
type Rules []*Rule

// some global vars
var (
	rulesCaches map[string]Rules
)

/*************************************************************
 * validation rule
 *************************************************************/

// Rule definition
type Rule struct {
	// eg "create" "update"
	scene string
	// need validate fields.
	fields string
	// is optional, only validate on value is not empty.
	optional bool
	// default value setting
	defValue interface{}
	// error message(s)
	message  string
	messages map[string]string
	// validator name, allow multi validators. eg "min", "range", "required;min(2)"
	validator string
	// arguments for the validator
	arguments []interface{}
	// some functions
	beforeFunc func(v *Validation) bool // func (val interface{}) bool
	filterFunc interface{}              // func (val interface{}) (newVal interface{})
	checkFunc  interface{}              // func (val interface{}, ...) bool
	// custom check is empty.
	emptyChecker func(val interface{}) bool
}

func NewRule(fields, validator string, args ...interface{}) *Rule {
	return &Rule{
		fields: fields,
		// args
		arguments: args,
		validator: validator,
	}
}

// With
func (r *Rule) With(fn func(r *Rule)) *Rule {
	fn(r)
	return r
}

// SetScene
func (r *Rule) SetScene(scene string) *Rule {
	r.scene = scene
	return r
}

// SetCheckFunc
func (r *Rule) SetCheckFunc(checkFunc interface{}) {
	r.checkFunc = checkFunc
}

// SetOptional
func (r *Rule) SetOptional(optional bool) *Rule {
	r.optional = optional
	return r
}

// SetMessage
func (r *Rule) SetMessage(errMsg string) *Rule {
	r.message = errMsg
	return r
}

// WithMessage
func (r *Rule) WithMessage(errMsg []string) *Rule {
	if len(errMsg) > 0 {
		r.message = errMsg[0]
	}

	return r
}

// SetMessages
func (r *Rule) SetMessages(msgMap SMap) *Rule {
	r.messages = msgMap
	return r
}

// Filters
func (r *Rule) UseFilters(names ...string) *Rule {
	// r.messages = msgMap
	return r
}

// FilterWithArgs
func (r *Rule) FilterWithArgs(name string, args ...interface{}) *Rule {
	// r.filterFunc = msgMap
	return r
}

func (r *Rule) Fields() []string {
	return stringSplit(r.fields, ",")
}

// Apply rule for the rule fields
func (r *Rule) Apply(v *Validation) bool {
	for _, field := range r.Fields() {
		// only one validator
		if !strings.ContainsRune(r.validator, ',') {
			r.Validate(field, r.validator, v)
		} else { // has multi validators
			vs := stringSplit(r.validator, ",")

			for _, validator := range vs {
				if r.Validate(field, validator, v) && v.StopOnError { // stop on error
					return true
				}
			}
		}

		if v.shouldStop() { // stop on error
			return true
		}
	}

	return false
}

// Validate the field by validator name
func (r *Rule) Validate(field, validator string, v *Validation) (ok bool) {
	// beforeFunc return false, skip validate
	if r.beforeFunc != nil && !r.beforeFunc(v) {
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

	switch validator {
	// case "min":
	// 	ok = Min(v.d.Get(field), r.arguments[0].(int64))
	// case "max":
	// 	ok = Max(v.d.MustInt64(field), r.arguments[0].(int64))
	// case "minLen", "minLength":
	// 	ok = MinLength(v.d.GetInt(field), r.arguments[0].(int))
	// case "maxLen", "maxLength":
	// 	ok = MaxLength(v.d.GetInt(field), r.arguments[0].(int))
	// case "range":
	// 	ok = v.required(field)
	// case "required":
	// 	ok = v.required(field)
	}

	return ok
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
		} else { // need convert type.
			argIn[i] = av.Convert(ft.In(i))
		}
	}

	// f.CallSlice()与Call() 不一样的是，参数的最后一个会被展开
	vs := fv.Call(argIn)
	return vs[0].Bool()
}

func panicf(format string, args ...interface{}) {
	panic("validate: " + fmt.Sprintf(format, args...))
}
