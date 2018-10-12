// Package validate is a generic go data validate library.
//
// Source code and other details for the project are available at GitHub:
//
// 	https://github.com/gookit/validate
//
package validate

import (
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
	// is optional, only validate on value is not empty. sometimes
	optional bool
	// default value setting
	defValue interface{}
	// error message
	message string
	// error messages, if fields contains multi field.
	// eg {
	// 	"field": "error message",
	// 	"field.validator": "error message",
	// }
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
	checkFunc     reflect.Value
	checkFuncMeta *funcMeta
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
	name := "rule." + r.fields
	fv := checkValidatorFunc(name, checkFunc)

	r.checkFunc = fv
	r.checkFuncMeta = newFuncMeta(name, false, fv)
	return r
}

// SetOptional only validate on value is not empty.
func (r *Rule) SetOptional(optional bool) *Rule {
	r.optional = optional
	return r
}

// SetMessage set error message.
// Usage:
// 	v.AddRule("name", "required").SetMessage("error message")
//
func (r *Rule) SetMessage(errMsg string) *Rule {
	r.message = errMsg
	return r
}

// SetMessages set error message map.
// Usage:
// 	v.AddRule("name,email", "required").SetMessages(MS{
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
	// scene name is not match. skip the rule
	if r.scene != "" && r.scene != v.scene {
		return false
	}

	// validate field value
	for _, field := range r.Fields() {
		if v.isNoNeedToCheck(field) {
			continue
		}

		// get field value.
		val, exist := v.Get(field)

		// empty value AND r.optional=true. skip check the field.
		if !exist && r.optional {
			continue
		}

		// apply filters func.
		if exist {
			val, err := applyFilters(val, r.filters, v)
			if err != nil { // has error
				v.AddError(filterError, err.Error())
				return true
			}

			// save filtered value.
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

	// call custom validator in the rule.
	fm := r.checkFuncMeta
	if fm == nil {
		name := ValidatorName(validator)
		fm = v.validatorMeta(name)
		if fm == nil {
			panicf("the validator '%s' is not exists", validator)
		}
	}

	// empty value AND skip on empty.
	if v.SkipOnEmpty && validator != "required" && IsEmpty(val) {
		return true
	}

	// some prepare and check.
	argNum := len(r.arguments) + 1 // "+1" is the "val" position
	rftVal := reflect.ValueOf(val)
	// check arg num is match
	fm.checkArgNum(argNum, validator)

	// convert val type, is first arg.
	ft := fm.fv.Type()
	firstTyp := ft.In(0).Kind()
	if firstTyp == rftVal.Kind() {
		ak, err := basicKind(rftVal)
		if err != nil { // todo check?
			return
		}

		// manual converted
		if nVal, _ := convertType(val, ak, rftVal.Kind()); nVal != nil {
			val = nVal
		}
	}

	// call built in validators
	ok = callValidator(v, fm, val, r.arguments)
	// build and collect error message
	if !ok {
		v.AddError(field, r.errorMessage(field, validator, v))
	}

	return
}

func (r *Rule) errorMessage(field, validator string, v *Validation) (msg string) {
	if r.messages != nil {
		var ok bool
		// use full key. "field.validator"
		fKey := field + "." + validator
		if msg, ok = r.messages[fKey]; ok {
			return
		}

		if msg, ok = r.messages[field]; ok {
			return
		}
	}

	if r.message != "" {
		return r.message
	}

	// built in error messages
	msg = v.trans.Message(validator, field, r.arguments...)

	return
}

func callValidator(v *Validation, fm *funcMeta, val interface{}, args []interface{}) (ok bool) {
	// 1. args data type convert

	ft := fm.fv.Type()
	lastTyp := reflect.Invalid
	lastArgIndex := fm.numIn - 1

	// isVariadic == true. last arg always is slice.
	// eg. "...int64" -> slice "[]int64"
	if fm.isVariadic {
		// get variadic kind. "[]int64" -> reflect.Int64
		lastTyp = getSliceItemKind(ft.In(lastArgIndex).String())
	}

	var wantTyp reflect.Kind

	// convert args data type
	for i, arg := range args {
		av := reflect.ValueOf(arg)

		// "+1" because first arg is val, need exclude it.
		if fm.isVariadic && i+1 >= lastArgIndex {
			if lastTyp == av.Kind() { // type is same
				continue
			}

			ak, err := basicKind(av)
			if err != nil {
				v.convertArgTypeError(fm.name, av.Kind(), wantTyp)
				return
			}

			if nVal, _ := convertType(args[i], ak, lastTyp); nVal != nil { // manual converted
				args[i] = nVal
				continue
			}

			// unable to convert
			v.convertArgTypeError(fm.name, av.Kind(), wantTyp)
			return
		}

		// "+1" because func first arg is val, need skip it.
		argITyp := ft.In(i + 1)
		wantTyp = argITyp.Kind()

		// type is same. or want type is interface
		if wantTyp == av.Kind() || wantTyp == reflect.Interface {
			continue
		}

		ak, err := basicKind(av)
		if err != nil {
			v.convertArgTypeError(fm.name, av.Kind(), wantTyp)
			return
		}

		if av.Type().ConvertibleTo(argITyp) { // can auto convert type.
			args[i] = av.Convert(argITyp).Interface()
		} else if nVal, _ := convertType(args[i], ak, wantTyp); nVal != nil { // manual converted
			args[i] = nVal
		} else { // unable to convert
			v.convertArgTypeError(fm.name, av.Kind(), wantTyp)
			return
		}
	}

	// fmt.Println(fm.name, val)
	// fmt.Printf("%#v\n", args)

	// 2. call built in validator
	switch fm.name {
	case "required":
		ok = v.Required(val)
	case "lt":
		ok = Lt(val, args[0].(int64))
	case "gt":
		ok = Gt(val, args[0].(int64))
	case "min":
		ok = Min(val, args[0].(int64))
	case "max":
		ok = Max(val, args[0].(int64))
	case "enum":
		ok = Enum(val, args[0])
	case "notIn":
		ok = NotIn(val, args[0])
	case "isInt":
		argLn := len(args)
		if argLn == 0 {
			ok = IsInt(val)
		} else if argLn == 1 {
			ok = IsInt(val, args[0].(int64))
		} else { // argLn == 2
			ok = IsInt(val, args[0].(int64), args[1].(int64))
		}
	case "isNumber":
		ok = IsNumber(val.(string))
	case "length":
		ok = Length(val, args[0].(int))
	case "minLength":
		ok = MinLength(val, args[0].(int))
	case "maxLength":
		ok = MaxLength(val, args[0].(int))
	case "regexp":
		ok = Regexp(val.(string), args[0].(string))
	case "between":
		ok = Between(val, args[0].(int64), args[1].(int64))
	case "isJSON":
		ok = IsJSON(val.(string))
	default: // is user custom validators
		ok = callValidatorValue(fm.fv, val, args)
	}

	return
}

func callValidatorValue(fv reflect.Value, val interface{}, args []interface{}) bool {
	argNum := len(args)

	// build params for the validator func.
	argIn := make([]reflect.Value, argNum+1)
	argIn[0] = reflect.ValueOf(val)

	for i := 0; i < argNum; i++ {
		argIn[i+1] = reflect.ValueOf(args[i])
	}

	// f.CallSlice()与Call() 不一样的是，CallSlice参数的最后一个会被展开
	// vs := fv.Call(argIn)
	return fv.Call(argIn)[0].Bool()
}
