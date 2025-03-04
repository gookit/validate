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

// ValidateErr do validate processing and return error
func (v *Validation) ValidateErr(scene ...string) error {
	if v.Validate(scene...) {
		return nil
	}
	return v.Errors
}

// ValidateE do validate processing and return Errors
//
// NOTE: need use len() to check the return is empty or not.
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

	// release instance to pool TODO
	// defer func() {
	// 	v.resetRules()
	// 	vPool.Put(v)
	// }()

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
	if v.hasError { // clear safe data on error.
		v.safeData = make(map[string]any)
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
	// validator name is not "requiredXXX"
	isNotRequired := r.nameNotRequired

	// apply filter func
	for _, field := range r.fields {
		val, _, _ := v.GetWithDefault(field)
		if r.filterFunc != nil {
			if val, err = r.filterFunc(val); err != nil {
				v.AddError(filterError, filterError, field+": "+err.Error())
				return true
			}
		}
		// save filtered value
		v.filteredData[field], err = v.updateValue(field, val)
		if err != nil {
			v.AddErrorf(field, err.Error())
			if v.StopOnError {
				return true
			}
			continue
		}
	}

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

		// get field value
		val, _, isDefault := v.GetWithDefault(field)

		// value not exists but has default value
		if isDefault {
			// update source data field value and re-set value
			val, err = v.updateValue(field, val)
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
		} else if r.optional { // r.optional=true. skip check.
			continue
		}

		// empty value AND is not required* AND skip on empty.
		if r.skipEmpty && isNotRequired && IsEmpty(val) {
			continue
		}

		// validate field value
		if r.valueValidate(field, name, val, v) {
			v.safeData[field] = val
		} else { // build and collect error message
			v.AddError(field, r.validator, r.errorMessage(field, r.validator, v))
		}

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

	ss := make([]string, 0, len(r.arguments))
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

// value by tryGet(key) TODO
type value struct {
	val any
	key string
	// has dot-star ".*" in the key. eg: details.sub.*.field
	dotStar bool
	// last index of dot-star on the key. eg: details.sub.*.field, lastIdx=11
	lastIdx int
	// is required or requiredXX check
	require bool
}

// validate the field value
//
//   - field: the field name. eg: "name", "details.sub.*.field"
//   - name: the validator name. eg: "required", "min"
func (r *Rule) valueValidate(field, name string, val any, v *Validation) (ok bool) {
	// "-" OR "safe" mark field value always is safe.
	if name == RuleSafe1 || name == RuleSafe {
		return true
	}

	// support check sub element in a slice list. eg: field=top.user.*.name
	dotStarNum := strings.Count(field, ".*")

	// perf: The most commonly used rule "required" - direct call v.Required()
	if name == RuleRequired && dotStarNum == 0 {
		return v.Required(field, val)
	}

	// call value validator in the rule.
	fm := r.checkFuncMeta
	if fm == nil {
		// fallback: get validator from global or validation
		fm = v.validatorMeta(name)
		if fm == nil {
			panicf("the validator '%s' does not exist", r.validator)
		}
	}

	// some prepare and check.
	argNum := len(r.arguments) + 2 // "+2" is the "data" and "val" position
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

	ft := fm.fv.Type() // type of check func
	arg1Kind := ft.In(1).Kind()

	// rftVal := reflect.Indirect(reflect.ValueOf(val))
	rftVal := reflect.ValueOf(val)
	valKind := rftVal.Kind()

	if valKind == reflect.Slice && dotStarNum > 0 {
		sliceLen, sliceCap := rftVal.Len(), rftVal.Cap()

		// if dotStarNum > 1, need flatten multi level slice with depth=dotStarNum.
		if dotStarNum > 1 {
			rftVal = flatSlice(rftVal, dotStarNum-1)
			sliceLen, sliceCap = rftVal.Len(), rftVal.Cap()
		}

		// check requiredXX validate - flatten multi level slice, count ".*" number.
		// TIP: if len < cap: not enough elements in the slice. use empty val call validator.
		if !r.nameNotRequired && sliceLen < sliceCap {
			return callValidator(v, fm, field, nil, r.arguments)
		}

		var subVal any
		// check each element in the slice.
		for i := 0; i < sliceLen; i++ {
			subRv := indirectInterface(rftVal.Index(i))
			subKind := subRv.Kind()

			// 1.1 convert field value type, is func first argument.
			if r.nameNotRequired && arg1Kind != reflect.Interface && arg1Kind != subKind {
				subVal, ok = convValAsFuncArg1Type(arg1Kind, subKind, subRv.Interface())
				if !ok {
					v.convArgTypeError(field, fm.name, subKind, arg1Kind, 1)
					return false
				}
			} else {
				if subRv.IsValid() {
					subVal = subRv.Interface()
				} else {
					subVal = nil
				}
			}

			// 2. call built in validator
			if !callValidator(v, fm, field, subVal, r.arguments) {
				return false
			}
		}

		return true
	}

	// 1 convert field value type, is func first argument.
	if r.nameNotRequired && arg1Kind != reflect.Interface && arg1Kind != valKind {
		val, ok = convValAsFuncArg1Type(arg1Kind, valKind, val)
		if !ok {
			v.convArgTypeError(field, fm.name, valKind, arg1Kind, 1)
			return false
		}
	}

	// 2. call built in validator
	return callValidator(v, fm, field, val, r.arguments)
}

// convert input field value type, is validator func first argument.
func convValAsFuncArg1Type(arg1Kind, valKind reflect.Kind, val any) (any, bool) {
	// If the validator function does not expect a pointer, but the value is a pointer,
	// dereference the value.
	if arg1Kind != reflect.Ptr && valKind == reflect.Ptr {
		if val == nil {
			return nil, true
		}

		val = reflect.ValueOf(val).Elem().Interface()
		valKind = reflect.TypeOf(val).Kind()
	}

	// ak, err := basicKind(rftVal)
	bk, err := basicKindV2(valKind)
	if err != nil {
		return nil, false
	}

	// manual converted
	if nVal, _ := convTypeByBaseKind(val, bk, arg1Kind); nVal != nil {
		return nVal, true
	}
	// TODO return nil, false
	return val, true
}

func callValidator(v *Validation, fm *funcMeta, field string, val any, args []any) (ok bool) {
	// use `switch` can avoid using reflection to call methods and improve speed
	// fm.name please see pkg var: validatorValues
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
		ok = callValidatorValue(v, fm.fv, val, args)
	}
	return
}

// convert args data type
func convertArgsType(v *Validation, fm *funcMeta, field string, args []any) (ok bool) {
	if len(args) == 0 {
		return true
	}

	ft := fm.fv.Type()
	lastTyp := reflect.Invalid
	lastArgIndex := fm.numIn - 1

	// fix: isVariadic == true. last arg always is slice.
	// eg: "...int64" -> slice "[]int64"
	if fm.isVariadic {
		// get variadic kind. "[]int64" -> reflect.Int64
		lastTyp = getVariadicKind(ft.In(lastArgIndex))
	}

	// only one args and type is any
	if lastArgIndex == 1 && lastTyp == reflect.Interface {
		return true
	}

	var wantKind reflect.Kind

	// convert args data type
	for i, arg := range args {
		av := reflect.ValueOf(arg)
		// index in the func
		// "+2" because func first arg is data and second arg is val. need skip it.
		fcArgIndex := i + 2
		argVKind := av.Kind()

		// Notice: "+2" because first arg is data and second arg is val, need exclude it.
		if fm.isVariadic && i+2 >= lastArgIndex {
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

		// "+2" because func first arg is data and second arg is val. need skip it.
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

func callValidatorValue(v *Validation, fv reflect.Value, val any, args []any) bool {
	// build params for the validator func.
	argNum := len(args)
	argIn := make([]reflect.Value, argNum+2)

	dataVal := reflect.ValueOf(v.FilteredData())
	if !dataVal.IsValid() {
		dataVal = nilRVal
	}
	// if val is any(nil): rftVal.IsValid()==false
	// if val is typed(nil): rftVal.IsValid()==true
	rftVal := reflect.ValueOf(val)
	// fix: #125 fv.Call() will panic on rftVal.Kind() is Invalid
	if !rftVal.IsValid() {
		rftVal = nilRVal
	}

	// Add this check to handle pointer values
	if rftVal.Kind() == reflect.Ptr && !rftVal.IsNil() {
		rftVal = rftVal.Elem()
	}

	argIn[0] = dataVal
	argIn[1] = rftVal
	for i := 0; i < argNum; i++ {
		rftValA := reflect.ValueOf(args[i])
		if !rftValA.IsValid() {
			rftValA = nilRVal
		}
		argIn[i+2] = rftValA
	}

	// TODO panic recover, refer the text/template/funcs.go
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		if e, ok := r.(error); ok {
	// 			err = e
	// 		} else {
	// 			err = fmt.Errorf("%v", r)
	// 		}
	// 	}
	// }()

	// NOTICE: f.CallSlice()与Call() 不一样的是，CallSlice参数的最后一个会被展开
	// vs := fv.Call(argIn)
	return fv.Call(argIn)[0].Bool()
}
