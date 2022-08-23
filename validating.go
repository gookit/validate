package validate

import (
	"reflect"
	"strings"
)

// const requiredValidator = "required"

// the validating result status:
// 0 ok 1 skip 2 fail
const (
	statusOk uint8 = iota
	statusSkip
	statusFail
)

/*************************************************************
 * Do Validating
 *************************************************************/

// ValidateData validate given data-source
func (v *Validation) ValidateData(data DataFace) bool {
	v.data = data
	return v.Validate()
}

// ValidateE do validate processing and return error
func (v *Validation) ValidateE(scene ...string) Errors {
	if v.Validate(scene...) {
		return nil
	}
	return v.Errors
}

// Validate processing
func (v *Validation) Validate(scene ...string) bool {
	// has been validated OR has error
	if v.hasValidated || v.shouldStop() {
		return v.IsSuccess()
	}

	// init scene info
	v.SetScene(scene...)
	v.sceneFields = v.sceneFieldMap()

	// apply filter rules before validate.
	if !v.Filtering() && v.StopOnError {
		return false
	}

	// apply rule to validate data.
	for _, rule := range v.rules {
		if rule.Apply(v) {
			break
		}
	}

	v.hasValidated = true
	if v.hasError {
		// clear safe data on error.
		v.safeData = make(map[string]interface{})
	}

	return v.IsSuccess()
}

// Apply current rule for the rule fields
func (r *Rule) Apply(v *Validation) (stop bool) {
	// scene name is not match. skip the rule
	if r.scene != "" && r.scene != v.scene {
		return
	}

	// has beforeFunc and it returns FALSE, skip validate
	if r.beforeFunc != nil && !r.beforeFunc(v) {
		return
	}

	var err error
	// get real validator name
	name := r.realName
	// validator name is not "required"
	isNotRequired := r.nameNotRequired

	// validate each field
	for _, field := range r.fields {
		if v.isNotNeedToCheck(field) {
			continue
		}

		// uploaded file validate
		if isFileValidator(name) {
			status := r.fileValidate(field, name, v)
			if status == statusFail {
				// build and collect error message
				v.AddError(field, r.validator, r.errorMessage(field, r.validator, v))
				if v.StopOnError {
					return true
				}
			}
			continue
		}

		// get field value. val, exist := v.Get(field)
		val, exist, isDefault := v.GetWithDefault(field)

		// value not exists but has default value
		if isDefault {
			// update source data field value and re-set value
			val, err := v.updateValue(field, val)
			if err != nil {
				v.AddErrorf(field, err.Error())
				if v.StopOnError {
					return true
				}
				continue
			}

			// dont need check default value
			if !v.CheckDefault {
				v.safeData[field] = val // save validated value.
				continue
			}

			// go on check custom default value
			exist = true
		} else if r.optional { // r.optional=true. skip check.
			continue
		}

		// apply filter func.
		if exist && r.filterFunc != nil {
			if val, err = r.filterFunc(val); err != nil {
				v.AddError(filterError, filterError, err.Error())
				return true
			}

			// update source field value
			newVal, err := v.updateValue(field, val)
			if err != nil {
				v.AddErrorf(field, err.Error())
				if v.StopOnError {
					return true
				}
				continue
			}

			// re-set value
			val = newVal
			// save filtered value.
			v.filteredData[field] = val
		}

		// empty value AND is not required* AND skip on empty.
		if r.skipEmpty && isNotRequired && IsEmpty(val) {
			continue
		}

		// validate field value
		if r.valueValidate(field, name, val, v) {
			v.safeData[field] = val // save validated value.
		} else { // build and collect error message
			v.AddError(field, r.validator, r.errorMessage(field, r.validator, v))
		}

		// stop on error
		if v.shouldStop() {
			return true
		}
	}

	return false
}

func (r *Rule) fileValidate(field, name string, v *Validation) uint8 {
	// check data source
	form, ok := v.data.(*FormData)
	if !ok {
		return statusFail
	}

	// skip on empty AND field not exist
	if r.skipEmpty && !form.HasFile(field) {
		return statusSkip
	}

	var ss []string
	for _, item := range r.arguments {
		ss = append(ss, item.(string))
	}

	switch name {
	case "isFile":
		ok = v.IsFormFile(form, field)
	case "isImage":
		ok = v.IsFormImage(form, field, ss...)
	case "inMimeTypes":
		if ln := len(ss); ln == 0 {
			panicf("not enough parameters for validator '%s'!", r.validator)
		} else if ln == 1 {
			//noinspection GoNilness
			ok = v.InMimeTypes(form, field, ss[0])
		} else { // ln > 1
			//noinspection GoNilness
			ok = v.InMimeTypes(form, field, ss[0], ss[1:]...)
		}
	}

	if ok {
		return statusOk
	}
	return statusFail
}

// validate the field value
func (r *Rule) valueValidate(field, name string, val interface{}, v *Validation) (ok bool) {
	// "-" OR "safe" mark field value always is safe.
	if name == "-" || name == "safe" {
		return true
	}

	// call custom validator in the rule.
	fm := r.checkFuncMeta
	if fm == nil {
		// get validator for global or validation
		fm = v.validatorMeta(name)
		if fm == nil {
			panicf("the validator '%s' does not exist", r.validator)
		}
	}

	// some prepare and check.
	argNum := len(r.arguments) + 1 // "+1" is the "val" position
	// check arg num is match, need exclude "requiredXXX"
	if r.nameNotRequired {
		//noinspection GoNilness
		fm.checkArgNum(argNum, r.validator)
	}

	// 1. args data type convert
	args := r.arguments
	if ok = convertArgsType(v, fm, field, args); !ok {
		return false
	}

	ft := fm.fv.Type()
	arg0Kind := ft.In(0).Kind()

	// rftVal := reflect.Indirect(reflect.ValueOf(val))
	rftVal := reflect.ValueOf(val)
	valKind := rftVal.Kind()

	// feat: support check sub element in a slice list. eg: field=names.*
	if valKind == reflect.Slice && strings.HasSuffix(field, ".*") {
		var subVal interface{}
		for i := 0; i < rftVal.Len(); i++ {
			subRv := rftVal.Index(i)
			subKind := subRv.Kind()
			// 1.1 convert field value type, is func first argument.
			if r.nameNotRequired && arg0Kind != reflect.Interface && arg0Kind != subKind {
				subVal, ok = convValAsFuncArg0Type(arg0Kind, subKind, subRv.Interface())
				if !ok {
					v.convArgTypeError(field, fm.name, subKind, arg0Kind, 0)
					return false
				}
			} else {
				subVal = subRv.Interface()
			}

			// 2. call built in validator
			if !callValidator(v, fm, field, subVal, r.arguments) {
				return false
			}
		}
		return true
	}

	// 1.1 convert field value type, is func first argument.
	if r.nameNotRequired && arg0Kind != reflect.Interface && arg0Kind != valKind {
		val, ok = convValAsFuncArg0Type(arg0Kind, valKind, val)
		if !ok {
			v.convArgTypeError(field, fm.name, valKind, arg0Kind, 0)
			return false
		}
	}

	// 2. call built in validator
	return callValidator(v, fm, field, val, r.arguments)
}

// convert input field value type, is validator func first argument.
func convValAsFuncArg0Type(arg0Kind, valKind reflect.Kind, val interface{}) (interface{}, bool) {
	// ak, err := basicKind(rftVal)
	bk, err := basicKindV2(valKind)
	if err != nil {
		return nil, false
	}

	// manual converted
	if nVal, _ := convTypeByBaseKind(val, bk, arg0Kind); nVal != nil {
		return nVal, true
	}

	// TODO return nil, false
	return val, true
}

func callValidator(v *Validation, fm *funcMeta, field string, val interface{}, args []interface{}) (ok bool) {
	// use `switch` can avoid using reflection to call methods and improve speed
	switch fm.name {
	case "required":
		ok = v.Required(field, val)
	case "requiredIf":
		ok = v.RequiredIf(field, val, args2strings(args)...)
	case "requiredUnless":
		ok = v.RequiredUnless(field, val, args2strings(args)...)
	case "requiredWith":
		ok = v.RequiredWith(field, val, args2strings(args)...)
	case "requiredWithAll":
		ok = v.RequiredWithAll(field, val, args2strings(args)...)
	case "requiredWithout":
		ok = v.RequiredWithout(field, val, args2strings(args)...)
	case "requiredWithoutAll":
		ok = v.RequiredWithoutAll(field, val, args2strings(args)...)
	case "lt":
		ok = Lt(val, args[0])
	case "gt":
		ok = Gt(val, args[0])
	case "min":
		ok = Min(val, args[0])
	case "max":
		ok = Max(val, args[0])
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
		ok = IsNumber(val)
	case "isStringNumber":
		ok = IsStringNumber(val.(string))
	case "length":
		ok = Length(val, args[0].(int))
	case "minLength":
		ok = MinLength(val, args[0].(int))
	case "maxLength":
		ok = MaxLength(val, args[0].(int))
	case "stringLength":
		if argLn := len(args); argLn == 1 {
			ok = RuneLength(val, args[0].(int))
		} else if argLn == 2 {
			ok = RuneLength(val, args[0].(int), args[1].(int))
		}
	case "regexp":
		ok = Regexp(val.(string), args[0].(string))
	case "between":
		ok = Between(val, args[0].(int64), args[1].(int64))
	case "isJSON":
		ok = IsJSON(val.(string))
	case "isSlice":
		ok = IsSlice(val)
	default:
		// 3. call user custom validators, will call by reflect
		ok = callValidatorValue(fm.fv, val, args)
	}
	return
}

// convert args data type
func convertArgsType(v *Validation, fm *funcMeta, field string, args []interface{}) (ok bool) {
	if len(args) == 0 {
		return true
	}

	ft := fm.fv.Type()
	lastTyp := reflect.Invalid
	lastArgIndex := fm.numIn - 1

	// fix: isVariadic == true. last arg always is slice.
	// eg. "...int64" -> slice "[]int64"
	if fm.isVariadic {
		// get variadic kind. "[]int64" -> reflect.Int64
		lastTyp = getVariadicKind(ft.In(lastArgIndex))
	}

	// only one args and type is interface{}
	if lastArgIndex == 1 && lastTyp == reflect.Interface {
		return true
	}

	var wantKind reflect.Kind

	// convert args data type
	for i, arg := range args {
		av := reflect.ValueOf(arg)
		// index in the func
		// "+1" because func first arg is `val`, need skip it.
		fcArgIndex := i + 1
		argVKind := av.Kind()

		// Notice: "+1" because first arg is field-value, need exclude it.
		if fm.isVariadic && i+1 >= lastArgIndex {
			if lastTyp == argVKind { // type is same
				continue
			}

			ak, err := basicKindV2(argVKind)
			if err != nil {
				v.convArgTypeError(field, fm.name, argVKind, wantKind, fcArgIndex)
				return
			}

			// manual converted
			if nVal, _ := convTypeByBaseKind(args[i], ak, lastTyp); nVal != nil {
				args[i] = nVal
				continue
			}

			// unable to convert
			v.convArgTypeError(field, fm.name, argVKind, wantKind, fcArgIndex)
			return
		}

		// "+1" because func first arg is val, need skip it.
		argIType := ft.In(fcArgIndex)
		wantKind = argIType.Kind()

		// type is same. or want type is interface
		if wantKind == argVKind || wantKind == reflect.Interface {
			continue
		}

		ak, err := basicKindV2(argVKind)
		if err != nil {
			v.convArgTypeError(field, fm.name, argVKind, wantKind, fcArgIndex)
			return
		}

		// can auto convert type.
		if av.Type().ConvertibleTo(argIType) {
			args[i] = av.Convert(argIType).Interface()
		} else if nVal, _ := convTypeByBaseKind(args[i], ak, wantKind); nVal != nil { // manual converted
			args[i] = nVal
		} else { // unable to convert
			v.convArgTypeError(field, fm.name, argVKind, wantKind, fcArgIndex)
			return
		}
	}

	return true
}

func callValidatorValue(fv reflect.Value, val interface{}, args []interface{}) bool {
	// build params for the validator func.
	argNum := len(args)
	argIn := make([]reflect.Value, argNum+1)

	// if val is interface{}(nil): rftVal.IsValid()==false
	// if val is typed(nil): rftVal.IsValid()==true
	rftVal := reflect.ValueOf(val)
	// fix: #125 fv.Call() will panic on rftVal.Kind() is Invalid
	if !rftVal.IsValid() {
		rftVal = nilRVal
	}

	argIn[0] = rftVal
	for i := 0; i < argNum; i++ {
		rftValA := reflect.ValueOf(args[i])
		if !rftValA.IsValid() {
			rftValA = nilRVal
		}
		argIn[i+1] = rftValA
	}

	// NOTICE: f.CallSlice()与Call() 不一样的是，CallSlice参数的最后一个会被展开
	// vs := fv.Call(argIn)
	return fv.Call(argIn)[0].Bool()
}
