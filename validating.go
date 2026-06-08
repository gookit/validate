package validate

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/validate/v2/internal/fieldval"
	"github.com/gookit/validate/v2/internal/reflectx"
	ivalidators "github.com/gookit/validate/v2/internal/validators"
)

// valToString coerces a field value to string for the string validators in the
// callValidator switch (regexp/isJSON/isStringNumber). It must NOT panic: when
// val is already a Go string it is returned byte-for-byte unchanged (fast path),
// and named string types (Kind == String, e.g. `type MyStr string`) are read via
// reflect so they no longer panic on the old val.(string) assertion. Other types
// are coerced via strutil.ToString. The bool reports whether a usable string was
// produced — false means the value can't be stringified, so the caller treats the
// validation as failed instead of asserting val.(string) (which would panic).
func valToString(val any) (string, bool) {
	if s, ok := val.(string); ok {
		return s, true
	}
	if val == nil {
		return "", false
	}
	if rv := reflect.ValueOf(val); rv.Kind() == reflect.String {
		return rv.String(), true
	}
	s, err := strutil.ToString(val)
	return s, err == nil
}

// fieldStr 取字段值的字符串形式:有载体(vfv!=nil ⇒ vfv.Src==val)时复用其缓存 RV(避免
// valToString 的二次 reflect.ValueOf),否则回退 valToString。两路对同输入字节级一致。
func fieldStr(vfv *fieldval.FieldValue, val any) (string, bool) {
	if vfv != nil {
		return vfv.String()
	}
	return valToString(val)
}

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

// ValidateR runs validation and returns the outcome as a *ValidResult that is
// decoupled from this instance. It is the primitive behind the top-level Check()
// and works on ANY configured instance (struct/map/programmatic builder).
//
// The result's Errors/safeData/filteredData are MOVED out of v (pointer
// hand-over, O(1) — not copied), then v is Released. So the returned result is
// safe to hold indefinitely while v may be reused by a pool. For a non-pooled v
// (default New/Struct/Map path) Release() is a no-op and v is simply discarded.
//
// After ValidateR the instance must NOT be used again (its result has been moved
// out and it may have been returned to a pool). When you only need the boolean /
// error face, use Validate() bool + v.Errors instead.
func (v *Validation) ValidateR(scene ...string) *ValidResult {
	v.Validate(scene...)

	// move (not copy) the result out of v into the standalone result object.
	r := &ValidResult{
		Errors:       v.Errors,
		safeData:     v.safeData,
		filteredData: v.filteredData,
	}
	// hand over ownership: nil the moved maps on v so Release()'s clear() leaves
	// them alone and the lazy-alloc chain rebuilds cleanly on the next reuse.
	v.Errors = nil
	v.safeData = nil
	v.filteredData = nil
	v.Release() // no-op unless v came from a pool (Factory / Check)
	return r
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
	if v.hasError && !v.skipCollect { // clear safe data on error (skip in CheckErr fast path).
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

	// get real validator name
	name := r.realName

	// validate each field
	for _, field := range r.fields {
		if r.applyField(field, name, v) {
			return true
		}
	}

	return false
}

// applyField runs the per-field validation steps in order, returning true when
// the whole rule should stop. The step order and side effects are identical to
// the original single-loop body of Rule.Apply; it is split out only so Apply
// itself is a thin scene/beforeFunc guard plus the per-field loop.
func (r *Rule) applyField(field, name string, v *Validation) (stop bool) {
	if v.isNotNeedToCheck(field) {
		return false
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
		return false
	}

	var err error

	// get field value. val, exist := v.Get(field)
	val, exist, isDefault := v.GetWithDefault(field)

	// value not exists but has default value
	if isDefault {
		// update source data field value and re-set value
		val, err = v.updateValue(field, val)
		if err != nil {
			v.AddErrorf(field, err.Error())
			if v.StopOnError {
				return true
			}
			return false
		}

		// dont need check default value
		if !v.CheckDefault {
			v.commitValue(field, val) // safeData 或 skipCollect 1 槽
			return false
		}

		// go on check custom default value
		exist = true
	} else if r.optional { // r.optional=true. skip check.
		return false
	}

	// apply filter func.
	if exist && r.filterFunc != nil {
		if val, err = r.filterFunc(val); err != nil {
			v.AddError(filterError, filterError, field+": "+err.Error())
			return true
		}

		// update source field value
		newVal, err := v.updateValue(field, val)
		if err != nil {
			v.AddErrorf(field, err.Error())
			if v.StopOnError {
				return true
			}
			return false
		}

		// re-set value
		val = newVal
		// save filtered value.
		if v.skipCollect {
			v.scKey, v.scVal = field, val
		} else {
			v.ensureFilteredData() // lazy
			v.filteredData[field] = val
		}
	}

	// empty value AND is not required* AND skip on empty.
	if r.skipEmpty && r.nameNotRequired && IsEmpty(val) {
		return false
	}

	// validate field value
	if r.valueValidate(field, name, val, v) {
		if v.data != nil && v.data.Type() == sourceForm {
			field, _, _ = strings.Cut(field, ".*")
		}
		v.commitValue(field, val) // safeData 或 skipCollect 1 槽
	} else { // build and collect error message
		msg := r.errorMessage(field, r.validator, v)
		// opt-in: append the failing value to the message (issue #184). default
		// off keeps the message byte-for-byte unchanged.
		if v.ErrShowValue {
			msg = fmt.Sprintf("%s (value: %v)", msg, val)
		}
		v.AddError(field, r.validator, msg)
	}

	if v.shouldStop() {
		return true
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
//
// This is the validation hot path. The large, tangled ".*" wildcard-slice
// branch is extracted into validateWildcardSlice for readability (it only runs
// for ".*" fields, off the common path); the rest is kept inline so the hot
// path's call-frame count and behavior are identical to before the split.
func (r *Rule) valueValidate(field, name string, val any, v *Validation) (ok bool) {
	// "-" OR "safe" mark field value always is safe.
	if name == RuleSafe1 || name == RuleSafe {
		return true
	}

	// T5: 自定义类型 → 提取底层值,使 required/empty/compare 都作用于提取值。
	// 门控 hasCustomTypes 内联在此:未注册时仅一次 atomic load 即短路,不进入
	// resolveCustomType 函数调用,保证热路径零开销。
	if hasCustomTypes.Load() {
		if ev, ok := resolveCustomType(val); ok {
			val = ev
		}
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

	// 1. args number check
	//goland:noinspection GoDfaNilDereference
	ft := fm.fv.Type() // type of check func
	valArgKind := ft.In(0).Kind()
	// if arg 0 is DataFace, need to add "data" to args.
	addNum := 1
	if ft.In(0) == dataFaceType {
		addNum += 1
		valArgKind = ft.In(1).Kind()
	}

	// R3: fieldctx 风格校验器签名固定为 func(FieldCtx)bool, 规则 args 经 fc.Arg() 取,
	// 不作为函数形参 → 跳过 checkArgNum(否则 argNum!=numIn=1 panic)与 convertArgsType
	// (否则会按 In(0)=接口错误转换);值类型转换块因 valArgKind 恒为 Interface 已天然跳过。
	isFieldCtx := fm.style == styleFieldCtx

	// some prepare and check.
	argNum := len(r.arguments) + addNum // "data" and "val" position
	// check arg num is match, need exclude "requiredXXX"
	if r.nameNotRequired && !isFieldCtx {
		//noinspection GoNilness
		fm.checkArgNum(argNum, r.validator)
	}

	// 2. args data type convert. Skip when the static template already
	// pre-converted these args at build time (P3a: r.argsReady).
	args := r.arguments
	if !r.argsReady && !isFieldCtx {
		if ok = convertArgsType(v, fm, field, args, addNum); !ok {
			return false
		}
	}

	// build the value carrier once; its rV() is computed lazily and reused
	// here and in the downstream callValidatorValue, removing the repeated
	// reflect.ValueOf(val) (痛点 A, design §4.3).
	//
	// NOTE: rV() substitutes nilRVal for an any(nil) src so the Call path won't
	// panic (#125). But valueValidate's original logic relied on valKind being
	// Invalid for nil; restore that so behavior is unchanged. rftVal itself is
	// only consumed in the Slice branch below, which nil never enters.
	fv := fieldval.New(field, val)
	rftVal := fv.RV()
	valKind := rftVal.Kind()
	if fv.Src == nil {
		valKind = reflect.Invalid
	}

	// ".*" wildcard slice branch: validate each element in the slice.
	if valKind == reflect.Slice && dotStarNum > 0 {
		return r.validateWildcardSlice(fm, field, rftVal, dotStarNum, valArgKind, addNum, v)
	}

	// 3. convert field value type, is func first argument.
	// vfv carries the original value; if a conversion happens below, val no
	// longer matches the carrier, so drop it (vfv=nil) to keep reflect correct.
	vfv := fv
	if r.nameNotRequired && !isFieldCtx && valArgKind != reflect.Interface && valArgKind != valKind {
		val, ok = convValAsFuncValArgType(valArgKind, valKind, val)
		if !ok {
			v.convArgTypeError(field, fm.name, valKind, valArgKind, 1)
			return false
		}
		vfv = nil
	}

	// 4. call built in validator
	return callValidator(v, fm, field, val, r.arguments, addNum, vfv)
}

// validateWildcardSlice validates the ".*" wildcard slice branch: it flattens
// multi-level slices, handles the requiredXX empty-slice / map parent-length
// cases, then converts and validates each element. Logic is平移 unchanged from
// the original inline branch.
//
// rftVal is the slice reflect.Value (fv.RV()); slice sub-elements never match
// the top-level carrier, so callValidator is always invoked with vfv=nil.
func (r *Rule) validateWildcardSlice(fm *funcMeta, field string, rftVal reflect.Value, dotStarNum int, valArgKind reflect.Kind, addNum int, v *Validation) (ok bool) {
	sliceLen := rftVal.Len()

	// if dotStarNum > 1, need flatten multi level slice with depth=dotStarNum.
	if dotStarNum > 1 {
		rftVal = flatSlice(rftVal, dotStarNum-1)
		sliceLen = rftVal.Len()
	}

	// check requiredXX validate - flatten multi level slice, count ".*" number.
	// TIP: if len == 0: no elements in the slice. use empty val call validator.
	// for map validation with wildcard, we need to compare with parent slice length.
	if !r.nameNotRequired && sliceLen == 0 {
		return callValidator(v, fm, field, nil, r.arguments, addNum, nil)
	}

	// for map validation with wildcard: check if some slice elements are missing fields
	// get the parent slice (before last .*) to compare lengths
	if !r.nameNotRequired && dotStarNum > 0 && v.data != nil && v.data.Type() == sourceMap {
		parentSliceLen := getParentSliceLen(field, v)
		if parentSliceLen > 0 && parentSliceLen > sliceLen {
			// parent slice has more elements than the returned values
			// means some elements are missing the field
			return callValidator(v, fm, field, nil, r.arguments, addNum, nil)
		}
	}

	var subVal any
	// check each element in the slice.
	for i := 0; i < sliceLen; i++ {
		subRv := indirectInterface(rftVal.Index(i))
		subKind := subRv.Kind()

		// T5: 自定义类型 → 提取每个元素的底层值,再走下面的类型转换/校验逻辑。
		// 门控内联:未注册时不进入提取分支,保证元素循环零开销。提取后用提取值的
		// reflect.Value 重置 subRv/subKind;提取为 nil 时 subRv 变为 invalid,直接
		// 当作 nil 处理,避免对 invalid Value 调用 Interface() 触发 panic。
		if hasCustomTypes.Load() && subRv.IsValid() {
			if ev, ok := resolveCustomType(subRv.Interface()); ok {
				if ev == nil {
					subVal = nil
					if !callValidator(v, fm, field, subVal, r.arguments, addNum, nil) {
						return false
					}
					continue
				}
				subRv = reflect.ValueOf(ev)
				subKind = subRv.Kind()
			}
		}

		// 1.1 convert field value type, is func first argument.
		if r.nameNotRequired && valArgKind != reflect.Interface && valArgKind != subKind {
			subVal, ok = convValAsFuncValArgType(valArgKind, subKind, subRv.Interface())
			if !ok {
				v.convArgTypeError(field, fm.name, subKind, valArgKind, 1)
				return false
			}
		} else {
			if subRv.IsValid() {
				subVal = subRv.Interface()
			} else {
				subVal = nil
			}
		}

		// 2. call built in validator. subVal is a slice element, not the
		// top-level value, so it gets no carrier (vfv=nil).
		if !callValidator(v, fm, field, subVal, r.arguments, addNum, nil) {
			return false
		}
	}

	return true
}

// convert input field value type, is validator func first argument.
func convValAsFuncValArgType(valArgKind, valKind reflect.Kind, val any) (any, bool) {
	// If the validator function does not expect a pointer, but the value is a pointer,
	// dereference the value.
	if valArgKind != reflect.Ptr && valKind == reflect.Ptr {
		if val == nil {
			return nil, true
		}

		val = reflect.ValueOf(val).Elem().Interface()
		valKind = reflect.TypeOf(val).Kind()
	}

	// manual converted
	if nVal, err := reflectx.ConvTypeByBaseKind(val, valArgKind); err == nil && nVal != nil {
		return nVal, true
	}

	return nil, false
}

// callValidator dispatches to the matching validator. The built-in switch
// avoids reflection; the default branch calls custom validators by reflection.
//
// vfv is the optional value carrier matching `val` (nil if val was transformed
// or is a slice sub-element); it is only forwarded to the reflective path.
func callValidator(v *Validation, fm *funcMeta, field string, val any, args []any, addNum int, vfv *fieldval.FieldValue) (ok bool) {
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
		if vfv != nil {
			ok = ivalidators.Lt(vfv, args[0])
		} else {
			ok = Lt(val, args[0])
		}
	case "gt":
		if vfv != nil {
			ok = ivalidators.Gt(vfv, args[0])
		} else {
			ok = Gt(val, args[0])
		}
	case "min":
		if vfv != nil {
			ok = ivalidators.Min(vfv, args[0])
		} else {
			ok = Min(val, args[0])
		}
	case "max":
		if vfv != nil {
			ok = ivalidators.Max(vfv, args[0])
		} else {
			ok = Max(val, args[0])
		}
	case "enum":
		if vfv != nil {
			ok = ivalidators.Enum(vfv, args[0])
		} else {
			ok = Enum(val, args[0])
		}
	case "rule_one_of": // #292: 列表参数同 enum, args[0] 为子校验器名 []string
		ok = v.RuleOneOf(val, args[0])
	case "notIn":
		if vfv != nil {
			ok = ivalidators.NotIn(vfv, args[0])
		} else {
			ok = NotIn(val, args[0])
		}
	case "isInt":
		if argLn := len(args); argLn == 0 {
			if vfv != nil {
				ok = ivalidators.IsInt(vfv)
			} else {
				ok = IsInt(val)
			}
		} else if argLn == 1 {
			if vfv != nil {
				ok = ivalidators.IsInt(vfv, args[0].(int64))
			} else {
				ok = IsInt(val, args[0].(int64))
			}
		} else { // argLn == 2
			if vfv != nil {
				ok = ivalidators.IsInt(vfv, args[0].(int64), args[1].(int64))
			} else {
				ok = IsInt(val, args[0].(int64), args[1].(int64))
			}
		}
	case "isString":
		if argLn := len(args); argLn == 0 {
			if vfv != nil {
				ok = ivalidators.IsString(vfv)
			} else {
				ok = IsString(val)
			}
		} else if argLn == 1 {
			if vfv != nil {
				ok = ivalidators.IsString(vfv, args[0].(int))
			} else {
				ok = IsString(val, args[0].(int))
			}
		} else { // argLn == 2
			if vfv != nil {
				ok = ivalidators.IsString(vfv, args[0].(int), args[1].(int))
			} else {
				ok = IsString(val, args[0].(int), args[1].(int))
			}
		}
	case "isNumber":
		ok = IsNumber(val)
	case "isStringNumber":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsStringNumber(s)
		}
	case "length":
		if vfv != nil {
			ok = ivalidators.Length(vfv, args[0].(int))
		} else {
			ok = Length(val, args[0].(int))
		}
	case "minLength":
		if vfv != nil {
			ok = ivalidators.MinLength(vfv, args[0].(int))
		} else {
			ok = MinLength(val, args[0].(int))
		}
	case "maxLength":
		if vfv != nil {
			ok = ivalidators.MaxLength(vfv, args[0].(int))
		} else {
			ok = MaxLength(val, args[0].(int))
		}
	case "stringLength":
		if argLn := len(args); argLn == 1 {
			ok = RuneLength(val, args[0].(int))
		} else if argLn == 2 {
			ok = RuneLength(val, args[0].(int), args[1].(int))
		}
	case "regexp":
		if s, sok := fieldStr(vfv, val); sok {
			ok = Regexp(s, args[0].(string))
		}
	case "between":
		if vfv != nil {
			ok = ivalidators.Between(vfv, args[0], args[1])
		} else {
			ok = Between(val, args[0], args[1])
		}
	case "isJSON":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsJSON(s)
		}
	case "isSlice":
		if vfv != nil {
			ok = ivalidators.IsSlice(vfv)
		} else {
			ok = IsSlice(val)
		}
	// R2.5a: 反射型类型校验器从 default(reflect.Call) 提升进 switch,改调 internal RV 版,
	// 消除 reflect.Call 开销 + argIn 分配。等价契约: ivalidators.X(c) ≡ public X(c.RealV().Interface())
	// (c 在 vfv==nil 时按 field+val 现造,复现 reflect.Call 的 vfv==nil 预解引用)。
	// c 仅被内部函数读取不存储,不逃逸。
	case "isBool":
		c := vfv
		if c == nil {
			c = fieldval.New(field, val)
		}
		ok = ivalidators.IsBool(c)
	case "isUint":
		c := vfv
		if c == nil {
			c = fieldval.New(field, val)
		}
		ok = ivalidators.IsUint(c)
	case "isArray":
		c := vfv
		if c == nil {
			c = fieldval.New(field, val)
		}
		if len(args) == 0 {
			ok = ivalidators.IsArray(c)
		} else { // strict 变参已被 convertArgsType 转为 bool
			ok = ivalidators.IsArray(c, args[0].(bool))
		}
	case "isMap":
		c := vfv
		if c == nil {
			c = fieldval.New(field, val)
		}
		ok = ivalidators.IsMap(c)
	case "isNumeric": // receives any, pass val directly
		ok = IsNumeric(val)
	// R2.5b: Contains/NotContains 提升进 switch 免 reflect.Call + argIn 分配。
	// 不搬 internal(依赖 includeElement→IsEqual 共享 root 助手),仍调 public,但传
	// c.RealV().Interface() 复现 reflect.Call 的 RealV 预解引用(指针容器解引用生效)。
	// args[0](sub)走单 any 形参,convertArgsType 不转换,与原 reflect.Call 路径一致。
	case "contains":
		c := vfv
		if c == nil {
			c = fieldval.New(field, val)
		}
		ok = Contains(c.RealV().Interface(), args[0])
	case "notContains":
		c := vfv
		if c == nil {
			c = fieldval.New(field, val)
		}
		ok = NotContains(c.RealV().Interface(), args[0])
	// --- single-arg string validators: T2 移入 switch,免反射 fv.Call ---
	// 统一用 fieldStr(vfv,val) 安全取字符串(命名字符串类型/可转换值都不 panic;有载体时
	// 复用其缓存 RV),取不到字符串时 ok 保持 false,行为与反射路径一致。
	case "isEmail":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsEmail(s)
		}
	case "isURL":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsURL(s)
		}
	case "isFullURL":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsFullURL(s)
		}
	case "isIP":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsIP(s)
		}
	case "isIPv4":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsIPv4(s)
		}
	case "isIPv6":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsIPv6(s)
		}
	case "isCIDR":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsCIDR(s)
		}
	case "isCIDRv4":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsCIDRv4(s)
		}
	case "isCIDRv6":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsCIDRv6(s)
		}
	case "isMAC":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsMAC(s)
		}
	case "isAlpha":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsAlpha(s)
		}
	case "isAlphaNum":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsAlphaNum(s)
		}
	case "isAlphaDash":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsAlphaDash(s)
		}
	case "isASCII":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsASCII(s)
		}
	case "isPrintableASCII":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsPrintableASCII(s)
		}
	case "isUUID":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsUUID(s)
		}
	case "isUUID3":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsUUID3(s)
		}
	case "isUUID4":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsUUID4(s)
		}
	case "isUUID5":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsUUID5(s)
		}
	case "isBase64":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsBase64(s)
		}
	case "isDataURI":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsDataURI(s)
		}
	case "isHexadecimal":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsHexadecimal(s)
		}
	case "isHexColor":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsHexColor(s)
		}
	case "isRGBColor":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsRGBColor(s)
		}
	case "isLatitude":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsLatitude(s)
		}
	case "isLongitude":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsLongitude(s)
		}
	case "isDNSName":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsDNSName(s)
		}
	case "isMultiByte":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsMultiByte(s)
		}
	case "isCnMobile":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsCnMobile(s)
		}
	case "isISBN10":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsISBN10(s)
		}
	case "isISBN13":
		if s, sok := fieldStr(vfv, val); sok {
			ok = IsISBN13(s)
		}
	case "hasWhitespace":
		if s, sok := fieldStr(vfv, val); sok {
			ok = HasWhitespace(s)
		}
	default:
		// 3. call user custom validators, will call by reflect (legacy)
		// or typed direct call (fieldctx style). dispatch inside.
		ok = callValidatorValue(v, fm, field, val, args, addNum, vfv)
	}
	return
}

// argConvError 携带 args 转换失败时上报错误所需的全部字段，
// 由纯函数 convertRuleArgs 返回，调用方据此决定如何上报。
type argConvError struct {
	field    string
	got      reflect.Kind // argVKind: 实参当前的 kind
	want     reflect.Kind // wantKind: 目标参数 kind
	argIndex int          // fcArgIndex: 在验证器函数签名中的参数下标
}

func (e *argConvError) Error() string {
	return fmt.Sprintf("cannot convert %s to arg#%d(%s)", e.got, e.argIndex, e.want)
}

// convertArgsType convert args data type. 薄封装：调用纯函数 convertRuleArgs，
// 失败时用 v.convArgTypeError 上报错误（字段/参数与原逻辑逐字一致）。
func convertArgsType(v *Validation, fm *funcMeta, field string, args []any, addNum int) (ok bool) {
	if err := convertRuleArgs(fm, field, args, addNum); err != nil {
		var ce *argConvError
		if errors.As(err, &ce) {
			v.convArgTypeError(ce.field, fm.name, ce.got, ce.want, ce.argIndex)
		}
		return false
	}
	return true
}

// convertRuleArgs 在不依赖 *Validation 的前提下，按验证器签名把 args 原地转换为目标类型。
// 成功返回 nil；失败返回携带 argVKind/wantKind/fcArgIndex 的 *argConvError，由调用方决定如何上报。
// 逻辑与原 convertArgsType 逐分支等价（含 isVariadic、单 any 早退、ConvertibleTo / reflectx.ConvTypeByBaseKind、nil 保留）。
func convertRuleArgs(fm *funcMeta, field string, args []any, addNum int) error {
	if len(args) == 0 {
		return nil
	}

	ft := fm.fv.Type()
	lastTyp := reflect.Invalid
	lastArgIndex := fm.numIn - 1

	// fix: isVariadic == true. last arg always is slice.
	// eg: "...int64" -> slice "[]int64"
	if fm.isVariadic {
		// get variadic kind. "[]int64" -> reflect.Int64
		lastTyp = reflectx.GetVariadicKind(ft.In(lastArgIndex))
	}

	// only one args and type is any
	if (lastArgIndex == 1 || (addNum == 2 && lastArgIndex == 2)) && lastTyp == reflect.Interface {
		return nil
	}

	var wantKind reflect.Kind

	// convert args data type
	for i, arg := range args {
		av := reflect.ValueOf(arg)
		// index in the func
		// "+addNum" because func first arg maybe data or val and second arg maybe val. need skip it.
		fcArgIndex := i + addNum
		argVKind := av.Kind()

		// Notice: "+addNum" because first arg maybe data or val and second arg maybe val, need exclude it.
		if fm.isVariadic && i+addNum >= lastArgIndex {
			if lastTyp == argVKind { // type is same
				continue
			}

			// manual converted
			if nVal, err := reflectx.ConvTypeByBaseKind(args[i], lastTyp); err == nil && nVal != nil {
				args[i] = nVal
				continue
			}

			// unable to convert. 注意：此分支沿用上一轮的 wantKind（与原逻辑一致）
			return &argConvError{field: field, got: argVKind, want: wantKind, argIndex: fcArgIndex}
		}

		argIType := ft.In(fcArgIndex)
		wantKind = argIType.Kind()

		// type is same. or want type is interface
		if wantKind == argVKind || wantKind == reflect.Interface {
			continue
		}

		// can auto convert type.
		if av.Type().ConvertibleTo(argIType) {
			args[i] = av.Convert(argIType).Interface()
		} else if nVal, err := reflectx.ConvTypeByBaseKind(args[i], wantKind); err == nil && nVal != nil { // manual converted
			args[i] = nVal
		} else { // unable to convert
			return &argConvError{field: field, got: argVKind, want: wantKind, argIndex: fcArgIndex}
		}
	}

	return nil
}

// callValidatorValue calls a custom validator by reflection.
//
// vfv is the optional value carrier matching `val`; when non-nil its cached
// reflect.Value is reused (rV/realV) to avoid re-doing reflect.ValueOf(val)
// and the pointer-deref here (痛点 A, design §4.3). It MUST be nil whenever
// `val` differs from the carrier's source (e.g. after type conversion or for
// slice sub-elements), so the reflect.Value stays consistent with `val`.
func callValidatorValue(v *Validation, fm *funcMeta, field string, val any, args []any, addNum int, vfv *fieldval.FieldValue) bool {
	// R3: fieldctx 风格 → typed 直调,免 reflect.Call 的 argIn 装箱。args 经 fc.Arg() 取。
	//
	// 注意(逃逸): 绝不把入参 vfv 指针存进会逃逸到堆的 fieldCtx,否则 escape 分析会把
	// vfv 形参标记为 leaking,连带让 legacy 热路径在 valueValidate 构造的 carrier 也
	// 逃逸到堆(实测 +5 allocs)。这里改为按 vfv 的(值拷贝)reflect.Value 现造一个新
	// carrier(NewRV 不重做 ValueOf),vfv 仅被读取不被存储,从而切断逃逸链。
	if fm.style == styleFieldCtx {
		var carrier *fieldval.FieldValue
		if vfv != nil {
			// 复用 vfv 已缓存的 RV(值拷贝传入),语义与 vfv.RealV()/Raw() 一致。
			carrier = fieldval.NewRV(field, vfv.RV())
		} else { // wildcard/转换值路径无载体,按 field+val 现造
			carrier = fieldval.New(field, val)
		}
		return fm.fcFunc(&fieldCtx{fv: carrier, field: field, args: args})
	}

	// ===== legacy reflect 路径(与改造前逐字节一致,仅 fv 改用 fm.fv) =====
	fv := fm.fv
	// build params for the validator func.
	argNum := len(args)
	argIn := make([]reflect.Value, argNum+addNum)

	var rftVal reflect.Value
	if vfv != nil {
		// reuse the carrier: rV() already substitutes nilRVal for any(nil)
		// (#125) and realV() applies the same non-nil pointer deref as below.
		rftVal = vfv.RealV()
	} else {
		// if val is any(nil): rftVal.IsValid()==false
		// if val is typed(nil): rftVal.IsValid()==true
		rftVal = reflect.ValueOf(val)
		// fix: #125 fv.Call() will panic on rftVal.Kind() is Invalid
		if !rftVal.IsValid() {
			rftVal = nilRVal
		}

		// Add this check to handle pointer values
		if rftVal.Kind() == reflect.Ptr && !rftVal.IsNil() {
			rftVal = rftVal.Elem()
		}
	}

	// if addNum == 1, means the first arg is val
	argIn[0] = rftVal
	// if addNum == 2, means the first arg is data and second arg is val
	if addNum == 2 {
		argIn[0] = reflect.ValueOf(v.data)
		argIn[1] = rftVal
	}
	for i := 0; i < argNum; i++ {
		rftValA := reflect.ValueOf(args[i])
		if !rftValA.IsValid() {
			rftValA = nilRVal
		}
		argIn[i+addNum] = rftValA
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

// getParentSliceLen get the length of parent slice before the last .*
// for path like "coding.*.details.cpt.*.encounter_uid", it will get the slice at "coding.*.details.cpt"
func getParentSliceLen(field string, v *Validation) int {
	// find the last .* position
	lastDotStarIdx := strings.LastIndex(field, ".*")
	if lastDotStarIdx == -1 {
		return 0
	}

	// get the parent path (before last .*)
	parentPath := field[:lastDotStarIdx]

	// get parent value - GetByPath returns different types depending on the path
	val, ok := maputil.GetByPath(parentPath, v.data.(*MapData).Map)
	if !ok || val == nil {
		return 0
	}

	// GetByPath can return different slice types:
	// - []any (for some paths)
	// - []map[string]any (for other paths)
	// - Nested []any containing inner slices (for wildcard paths)

	// Try []map[string]any first (common case for nested paths)
	if flatSlice, ok := val.([]map[string]any); ok {
		return len(flatSlice)
	}

	// Try []any
	outerSlice, ok := val.([]any)
	if !ok {
		return 0
	}

	// Check if it's a flat slice (elements are maps, not slices)
	if len(outerSlice) > 0 {
		_, isSlice := outerSlice[0].([]any)
		if !isSlice {
			// Flat slice - each element is a map
			if _, ok := outerSlice[0].(map[string]any); ok {
				return len(outerSlice)
			}
		}
	}

	// Handle nested slice case (parent path has wildcard)
	// Count total elements across all inner slices
	total := 0
	for _, item := range outerSlice {
		// Handle different slice types
		switch inner := item.(type) {
		case []any:
			total += len(inner)
		case []map[string]any:
			total += len(inner)
		default:
			// Try reflection for other types
			rv := reflect.ValueOf(item)
			if rv.Kind() == reflect.Slice {
				total += rv.Len()
			}
		}
	}

	return total
}
