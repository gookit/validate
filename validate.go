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

/*************************************************************
 * validation rule
 *************************************************************/

// Rule definition
type Rule struct {
	// eg "create" "update"
	scene string
	// need validate fields. allow multi.
	fields []string
	// is optional, only validate on value is not empty. sometimes
	optional bool
	// skip validate not exist field/empty value
	skipEmpty bool
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
	// validator name, allow multi validators. eg "min", "range", "required"
	validator string
	// arguments for the validator
	arguments []interface{}
	// some functions
	beforeFunc func(field string, v *Validation) bool // func (val interface{}) bool
	filterFunc func(val interface{}) (interface{}, error)
	// custom check func's mate info
	checkFuncMeta *funcMeta
	// custom check is empty.
	emptyChecker func(val interface{}) bool
}

// NewRule instance
func NewRule(fields, validator string, args ...interface{}) *Rule {
	return &Rule{
		fields: stringSplit(fields, ","),
		// validator args
		arguments: args,
		validator: validator,
	}
}

// SetScene name for the rule.
func (r *Rule) SetScene(scene string) *Rule {
	r.scene = scene
	return r
}

// SetOptional only validate on value is not empty.
func (r *Rule) SetOptional(optional bool) {
	r.optional = optional
}

// SetSkipEmpty skip validate not exist field/empty value
func (r *Rule) SetSkipEmpty(skipEmpty bool) {
	r.skipEmpty = skipEmpty
}

// SetCheckFunc set custom validate func.
func (r *Rule) SetCheckFunc(checkFunc interface{}) *Rule {
	var name string
	if r.validator != "" {
		name = "rule_" + r.validator
	} else {
		name = "rule_" + strings.Join(r.fields, "_")
	}

	fv := checkValidatorFunc(name, checkFunc)
	r.checkFuncMeta = newFuncMeta(name, false, fv)
	return r
}

// SetFilterFunc for the rule
func (r *Rule) SetFilterFunc(fn func(val interface{}) (interface{}, error)) *Rule {
	r.filterFunc = fn
	return r
}

// SetBeforeFunc for the rule. will call it before validate.
func (r *Rule) SetBeforeFunc(fn func(field string, v *Validation) bool) {
	r.beforeFunc = fn
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

// Fields field names list
func (r *Rule) Fields() []string {
	return r.fields
}

// Apply rule for the rule fields
func (r *Rule) Apply(v *Validation) (stop bool) {
	// scene name is not match. skip the rule
	if r.scene != "" && r.scene != v.scene {
		return false
	}

	var err error
	name := ValidatorName(r.validator)

	// validate each field
	for _, field := range r.fields {
		if v.isNoNeedToCheck(field) {
			continue
		}

		// uploaded file check
		if isFileValidator(name) {
			// build and collect error message
			if !r.fileValidate(field, name, v) {
				v.AddError(field, r.errorMessage(field, r.validator, v))
				// stop on error
				if v.StopOnError {
					return true
				}
			}

			continue
		}

		// get field value.
		val, exist := v.Get(field)
		if !exist && r.optional { // not exist AND r.optional=true. skip check.
			continue
		}

		// apply filter func.
		if exist && r.filterFunc != nil {
			if val, err = r.filterFunc(val); err != nil { // has error
				v.AddError(filterError, err.Error())
				return true
			}

			// save filtered value.
			v.filteredData[field] = val
		}

		if r.valueValidate(field, name, val, v) {
			v.safeData[field] = val // save validated value.
		} else { // build and collect error message
			v.AddError(field, r.errorMessage(field, r.validator, v))
		}

		// stop on error
		if v.shouldStop() {
			return true
		}
	}

	return false
}

func (r *Rule) fileValidate(field, name string, v *Validation) (ok bool) {
	// beforeFunc return false, skip validate
	if r.beforeFunc != nil && !r.beforeFunc(field, v) {
		return false
	}

	fd, ok := v.data.(*FormData)
	if !ok {
		return
	}

	// skip on empty AND field not exist
	if v.SkipOnEmpty && !fd.HasFile(field) {
		return true
	}

	var ss []string
	for _, item := range r.arguments {
		ss = append(ss, item.(string))
	}

	switch name {
	case "isFile":
		ok = v.IsFile(fd, field)
	case "isImage":
		ok = v.IsImage(fd, field, ss...)
	case "inMimeTypes":
		if ln := len(ss); ln == 0 {
			return false
		} else if ln == 1 {
			ok = v.InMimeTypes(fd, field, ss[0])
		} else {
			ok = v.InMimeTypes(fd, field, ss[0], ss[1:]...)
		}
	}

	return
}

// validate the field value
func (r *Rule) valueValidate(field, name string, val interface{}, v *Validation) (ok bool) {
	// "-" OR "safe" mark field value always is safe.
	if name == "-" || name == "safe" {
		return true
	}

	// beforeFunc return false, skip validate
	if r.beforeFunc != nil && !r.beforeFunc(field, v) {
		return false
	}

	// call custom validator in the rule.
	fm := r.checkFuncMeta
	if fm == nil {
		fm = v.validatorMeta(name)
		if fm == nil {
			panicf("the validator '%s' is not exists", r.validator)
		}
	}

	// empty value AND skip on empty.
	isNotRequired := name != "required"
	if v.SkipOnEmpty && isNotRequired && IsEmpty(val) {
		return true
	}

	// some prepare and check.
	argNum := len(r.arguments) + 1 // "+1" is the "val" position
	rftVal := reflect.ValueOf(val)
	valKind := rftVal.Kind()
	// check arg num is match
	if isNotRequired { // need exclude "required"
		fm.checkArgNum(argNum, r.validator)

		// convert val type, is first arg.
		ft := fm.fv.Type()
		firstTyp := ft.In(0).Kind()
		if firstTyp != valKind && firstTyp != reflect.Interface {
			ak, err := basicKind(rftVal)
			if err != nil { // todo check?
				v.convertArgTypeError(fm.name, valKind, firstTyp)
				return
			}

			// manual converted
			if nVal, _ := convertType(val, ak, firstTyp); nVal != nil {
				val = nVal
			}
		}
	}

	// call built in validators
	ok = callValidator(v, fm, field, val, r.arguments)
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
	return v.trans.Message(validator, field, r.arguments...)
}

// convert args data type
func convertArgsType(v *Validation, fm *funcMeta, args []interface{}) (ok bool) {
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

	return true
}

func callValidator(v *Validation, fm *funcMeta, field string, val interface{}, args []interface{}) (ok bool) {
	// 1. args data type convert
	if ok = convertArgsType(v, fm, args); !ok {
		return
	}

	// fmt.Println(fm.name, val)
	// fmt.Printf("%#v\n", args)

	// 2. call built in validator
	switch fm.name {
	case "required":
		ok = v.Required(field, val)
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
		if argLn := len(args); argLn == 0 {
			ok = IsInt(val)
		} else if argLn == 1 {
			ok = IsInt(val, args[0].(int64))
		} else { // argLn == 2
			ok = IsInt(val, args[0].(int64), args[1].(int64))
		}
	case "isString":
		if argLn := len(args); argLn == 0 {
			ok = IsString(val)
		} else if argLn == 1 {
			ok = IsString(val, args[0].(int))
		} else { // argLn == 2
			ok = IsString(val, args[0].(int), args[1].(int))
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
	default:
		// 3. call user custom validators
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
