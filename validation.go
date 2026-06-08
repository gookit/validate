package validate

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

// some default value settings.
const (
	fieldTag  = "json"
	filterTag = "filter"
	labelTag  = "label"

	messageTag  = "message"
	validateTag = "validate"

	filterError   = "_filter"
	validateError = "_validate"

	// sniff Length, use for detect file mime type
	sniffLen = 512
	// 32 MB
	defaultMaxMemory int64 = 32 << 20

	// validator type
	validatorTypeBuiltin int8 = 1
	validatorTypeCustom  int8 = 2
)

// Validation definition
type Validation struct {
	// pool is the owning sync.Pool when this instance was obtained from an
	// opt-in Factory (see factory.go). nil for the default New/Struct/Map path,
	// so Release() is a no-op there. Set by Factory.* and cleared on Release().
	pool *sync.Pool

	// source input data
	data DataFace
	// sd is a reusable StructData carried by POOLED instances (Factory / Check) so
	// a struct validation does not allocate a new StructData + fieldNames map on
	// every call. nil on the default New/Struct/Map path. It is reset (source
	// unbound, caches cleared) on reuse — see resetForReuse + StructData.fromStruct.
	sd *StructData
	// all validated fields list
	// fields []string

	// save filtered/validated safe data
	safeData M
	// filtered clean data
	filteredData M
	// save user custom set default values
	defValues map[string]any

	// Errors for validate
	Errors Errors
	// CacheKey for cache rules
	// CacheKey string
	// StopOnError If true: An error occurs, it will cease to continue to verify
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty
	SkipOnEmpty bool
	// UpdateSource Whether to update source field value, useful for struct validate
	UpdateSource bool
	// CheckDefault Whether to validate the default value set by the user
	CheckDefault bool
	// ErrShowValue Whether to append the failing value to the error message.
	// opt-in, copied from gOpt. see GitHub issue #184.
	ErrShowValue bool
	// CachingRules switch. default is False
	// CachingRules bool

	// mark has error occurs
	hasError bool
	// mark is filtered
	hasFiltered bool
	// mark is validated
	hasValidated bool
	// validate rules for the validation
	rules []*Rule

	// validators for the validation. map-value: 1=builtin, 2=custom
	validators map[string]int8
	// validator func meta info
	validatorMetas map[string]*funcMeta

	// current scene name
	scene string
	// scenes config.
	// {
	// 	"create": {"field0", "field1"}
	// 	"update": {"field0", "field2"}
	// }
	scenes SValues
	// should check fields in current scene.
	sceneFields map[string]uint8
	// scene fields that carry a ".*" wildcard (eg "Tags.*.Id"); matched against the
	// indexed rule names generated for slice elements (eg "Tags.0.Id"). (#283)
	sceneWildcards map[string]uint8

	// filtering rules for the validation
	filterRules []*FilterRule
	// filter func reflect.Value map
	filterValues map[string]reflect.Value

	// translator instance
	trans *Translator
	// optional fields, useful for sub-struct field in struct data. eg: "Parent"
	//
	// key is field name, value is field vale is: init=0 empty=1 not-empty=2.
	optionals map[string]int8

	// CheckErr(skipCollect) 模式状态。skipCollect=true 时跳过 safeData/filteredData
	// 收集,改用 scKey/scVal 1 槽缓存对"同字段连续取值"做装箱去重(镜像 safeData 的
	// 去重职责)。详见 docs/perf/checkerr-impl-plan.md。
	skipCollect bool
	scKey       string
	scVal       any
}

// NewEmpty new validation instance, but not with data.
func NewEmpty(scene ...string) *Validation {
	return NewValidation(nil, scene...)
}

// NewValidation new validation instance
func NewValidation(data DataFace, scene ...string) *Validation {
	return newValidation(data).SetScene(scene...)
}

/*************************************************************
 * lazy map allocation guards (perf Step 2)
 *
 * Each per-instance map in newEmpty() is allocated on FIRST WRITE only. Writing
 * to a nil map panics, so every write site must call its ensure*() first; reads
 * (range/len/index-get/comma-ok) are nil-safe and need no guard.
 *************************************************************/

func (v *Validation) ensureErrors() {
	if v.Errors == nil {
		v.Errors = make(Errors)
	}
}

func (v *Validation) ensureSafeData() {
	if v.safeData == nil {
		v.safeData = make(map[string]any)
	}
}

func (v *Validation) ensureFilteredData() {
	if v.filteredData == nil {
		v.filteredData = make(map[string]any)
	}
}

// commitValue records a field's validated value: into the skipCollect 1-slot
// cache (CheckErr fast path) or into safeData (normal collect path).
func (v *Validation) commitValue(field string, val any) {
	if v.skipCollect {
		v.scKey, v.scVal = field, val
		return
	}
	v.ensureSafeData()
	v.safeData[field] = val
}

func (v *Validation) ensureOptionals() {
	if v.optionals == nil {
		v.optionals = make(map[string]int8)
	}
}

func (v *Validation) ensureValidatorMaps() {
	if v.validators == nil {
		v.validators = make(map[string]int8)
	}
	if v.validatorMetas == nil {
		v.validatorMetas = make(map[string]*funcMeta)
	}
}

/*************************************************************
 * validation settings
 *************************************************************/

// ResetResult reset the validate result.
func (v *Validation) ResetResult() {
	// Step 2: result maps reset to nil (lazily re-allocated on first write), so a
	// Reset()'d instance keeps the no-alloc property on the next clean validation.
	v.Errors = nil
	v.hasError = false
	v.hasFiltered = false
	v.hasValidated = false
	// result data
	v.safeData = nil
	v.filteredData = nil
}

// Reset the Validation instance.
//
// Will resets:
//   - validate result
//   - validate rules
//   - validate filterRules
//   - custom validators TODO
func (v *Validation) Reset() {
	v.ResetResult()

	// v.validators = make(map[string]int8)
	v.resetRules()
}

func (v *Validation) resetRules() {
	// reset rules
	v.rules = v.rules[:0]
	v.optionals = nil // lazily re-allocated on first write (ensureOptionals)
	v.filterRules = v.filterRules[:0]
}

// resetForReuse fully resets ALL per-validation state back to the newEmpty()
// initial values, so a pooled instance can be safely reused for a different
// data source/type without any cross-validation data leak.
//
// This is intentionally a separate method from Reset()/ResetResult() (which only
// clear the result + rules and are part of the public, default-path API). It is
// only used by the opt-in Factory (factory.go) and Release(). Every field set in
// newEmpty()/NewValidation()/Create() that can be mutated during a validation is
// restored here. The pool field itself is NOT touched (Release manages it).
func (v *Validation) resetForReuse() {
	// --- source input ---
	v.data = nil
	// pooled StructData: unbind source + clear field caches but KEEP the
	// allocation for reuse. Prevents a pooled instance from pinning the last
	// validated struct while it sits idle in the pool.
	if v.sd != nil {
		v.sd.reset()
	}

	// --- result data + flags (mirrors ResetResult, but clears maps in place to
	// reuse the already-allocated buckets — this is the whole point of pooling) ---
	clear(v.Errors)
	v.hasError = false
	v.hasFiltered = false
	v.hasValidated = false
	clear(v.safeData)
	clear(v.filteredData)
	// user custom default values (lazily allocated, see SetDefValue)
	clear(v.defValues)

	// --- config flags: restore to global defaults (newEmpty uses gOpt) ---
	// NOTE: Struct() sets UpdateSource=true after Create; CheckDefault may be
	// toggled by callers. All must go back to the New-time initial values.
	v.StopOnError = gOpt.StopOnError
	v.SkipOnEmpty = gOpt.SkipOnEmpty
	v.ErrShowValue = gOpt.ErrShowValue
	v.UpdateSource = false
	v.CheckDefault = false

	// --- rules / filter rules / optionals (mirrors resetRules; keep cap) ---
	v.rules = v.rules[:0]
	v.filterRules = v.filterRules[:0]
	clear(v.optionals)

	// --- validators: drop per-type custom validators + lazily-bound ctx metas.
	// newEmpty() starts with empty maps; ctx validators rebind lazily to this
	// same v on next lookup (validatorMeta), so clearing is correct & required
	// (a struct's own FuncValue / AddValidator entries are type-specific). ---
	clear(v.validators)
	clear(v.validatorMetas)
	// instance-level custom filter funcs (lazily allocated, see AddFilter)
	clear(v.filterValues)

	// --- scene state ---
	v.scene = ""
	v.scenes = nil
	v.sceneFields = nil
	v.sceneWildcards = nil

	// --- translator: reset custom messages/labels/field-map back to empty.
	// Clear in place (matches Translator.Reset semantics: messages=nil custom
	// only, label/field maps emptied) to avoid 2 map allocs per reuse. ---
	clear(v.trans.messages)
	v.trans.messages = nil
	clear(v.trans.labelMap)
	clear(v.trans.fieldMap)

	// --- CheckErr(skipCollect) 状态:必须清,否则 CheckErr 用过的池实例被 Check
	// 复用时会残留 skipCollect=true 导致 Check 收不到 safeData。 ---
	v.skipCollect = false
	v.scKey = ""
	v.scVal = nil
}

// TODO Config(opt *Options) *Validation

// WithSelf config the Validation instance. TODO rename to WithConfig
func (v *Validation) WithSelf(fn func(v *Validation)) *Validation {
	fn(v)
	return v
}

// WithTrans with a custom translator
func (v *Validation) WithTrans(trans *Translator) *Validation {
	v.trans = trans
	return v
}

// WithScenarios is alias of the WithScenes()
func (v *Validation) WithScenarios(scenes SValues) *Validation {
	return v.WithScenes(scenes)
}

// WithScenes set scene config.
//
// Usage:
//
//	v.WithScenes(SValues{
//		"create": []string{"name", "email"},
//		"update": []string{"name"},
//	})
//	ok := v.AtScene("create").Validate()
func (v *Validation) WithScenes(scenes map[string][]string) *Validation {
	v.scenes = scenes
	return v
}

// AtScene setting current validate scene.
func (v *Validation) AtScene(scene string) *Validation {
	v.scene = scene
	return v
}

// InScene alias of the AtScene()
func (v *Validation) InScene(scene string) *Validation {
	return v.AtScene(scene)
}

// SetScene alias of the AtScene()
func (v *Validation) SetScene(scene ...string) *Validation {
	if len(scene) > 0 {
		v.AtScene(scene[0])
	}
	return v
}

/*************************************************************
 * add validators for validation
 *************************************************************/

// AddValidators to the Validation instance.
func (v *Validation) AddValidators(m map[string]any) *Validation {
	for name, checkFunc := range m {
		v.AddValidator(name, checkFunc)
	}
	return v
}

// AddValidator to the Validation instance. checkFunc must return a bool.
//
// Usage:
//
//	v.AddValidator("myFunc", func(data validate.DataFace, val any) bool {
//		// do validate val ...
//		return true
//	})
func (v *Validation) AddValidator(name string, checkFunc any) *Validation {
	fv := checkValidatorFunc(name, checkFunc)

	v.ensureValidatorMaps() // lazy
	v.validators[name] = validatorTypeCustom
	// v.validatorValues[name] = fv
	v.validatorMetas[name] = newFuncMeta(name, false, fv)

	return v
}

// ValidatorMeta get by name. get validator from global or validation instance.
func (v *Validation) validatorMeta(name string) *funcMeta {
	// current validation
	if fm, ok := v.validatorMetas[name]; ok {
		return fm
	}

	// from global validators
	if fm, ok := validatorMetas[name]; ok {
		return fm
	}

	// lazy-build a build-in context validator on first lookup (perf P5b).
	// binds the real v so both the switch-direct path (required-family, uses
	// only fm.Type()) and the reflect Call path (eqField/file..., needs the
	// receiver) behave identically to the previous eager construction.
	if builder, ok := ctxValidatorBuilders[name]; ok {
		fm := newFuncMeta(name, true, builder(v))
		v.ensureValidatorMaps() // lazy
		v.validators[name] = validatorTypeBuiltin
		v.validatorMetas[name] = fm
		return fm
	}

	// if v.data is StructData instance.
	if v.data.Type() == sourceStruct {
		fv, ok := v.data.(*StructData).FuncValue(name)
		if ok {
			fm := newFuncMeta(name, false, fv)
			// storage it.
			v.ensureValidatorMaps() // lazy
			v.validators[name] = validatorTypeCustom
			v.validatorMetas[name] = fm

			return fm
		}
	}
	return nil
}

// HasValidator check
func (v *Validation) HasValidator(name string) bool {
	name = ValidatorName(name)

	// current validation
	if _, ok := v.validatorMetas[name]; ok {
		return true
	}

	// build-in context validators are always available (bound lazily).
	if _, ok := ctxValidatorBuilders[name]; ok {
		return true
	}

	// global validators
	_, ok := validatorMetas[name]
	return ok
}

// Validators get all validator names
func (v *Validation) Validators(withGlobal bool) map[string]int8 {
	mp := make(map[string]int8, len(v.validators)+len(ctxValidatorBuilders))

	if withGlobal {
		for name, typ := range validators {
			mp[name] = typ
		}
	}

	// include the build-in context validators (always available, bound
	// lazily so they may not yet be present in v.validators after P5b).
	for name := range ctxValidatorBuilders {
		mp[name] = validatorTypeBuiltin
	}

	// instance validators last: already-built ctx + custom override above.
	for name, typ := range v.validators {
		mp[name] = typ
	}
	return mp
}

/*************************************************************
 * Do filtering/sanitize
 *************************************************************/

// Sanitize data by filter rules
func (v *Validation) Sanitize() bool { return v.Filtering() }

// Filtering data by filter rules
func (v *Validation) Filtering() bool {
	if v.hasFiltered {
		return v.IsSuccess()
	}

	// apply rule to validate data.
	for _, rule := range v.filterRules {
		if err := rule.Apply(v); err != nil { // has error
			v.AddError(filterError, filterError, rule.fields[0]+": "+err.Error())
			break
		}
	}

	v.hasFiltered = true
	return v.IsSuccess()
}

/*************************************************************
 * errors messages
 *************************************************************/

// WithTranslates settings. you can be custom field translates.
//
// Usage:
//
//		v.WithTranslates(map[string]string{
//			"name": "Username",
//			"pwd": "Password",
//	 })
func (v *Validation) WithTranslates(m map[string]string) *Validation {
	v.trans.AddLabelMap(m)
	return v
}

// AddTranslates settings data. like WithTranslates()
func (v *Validation) AddTranslates(m map[string]string) {
	v.trans.AddLabelMap(m)
}

// WithMessages settings. you can custom validator error messages.
//
// Usage:
//
//		// key is "validator" or "field.validator"
//		v.WithMessages(map[string]string{
//			"require": "oh! {field} is required",
//			"range": "oh! {field} must be in the range %d - %d",
//	 })
func (v *Validation) WithMessages(m map[string]string) *Validation {
	v.trans.AddMessages(m)
	return v
}

// AddMessages settings data. like WithMessages()
func (v *Validation) AddMessages(m map[string]string) {
	v.trans.AddMessages(m)
}

// WithError add error of the validation
func (v *Validation) WithError(err error) *Validation {
	if err != nil {
		v.AddError(validateError, validateError, err.Error())
	}
	return v
}

// AddError message for a field
func (v *Validation) AddError(field, validator, msg string) {
	if !v.hasError {
		v.hasError = true
	}

	v.ensureErrors() // lazy: only the error path allocates Errors
	field = v.trans.FieldName(field)
	v.Errors.Add(field, validator, msg)
}

// AddErrorf add a formatted error message
func (v *Validation) AddErrorf(field, msgFormat string, args ...any) {
	v.AddError(field, validateError, fmt.Sprintf(msgFormat, args...))
}

// Trans get translator
func (v *Validation) Trans() *Translator {
	// if v.trans == nil {
	// 	v.trans = StdTranslator
	// }
	return v.trans
}

func (v *Validation) convArgTypeError(field, name string, argKind, wantKind reflect.Kind, argIdx int) {
	v.AddErrorf(field, "cannot convert %s to arg#%d(%s), validator '%s'", argKind, argIdx, wantKind, name)
}

/*************************************************************
 * getter methods
 *************************************************************/

// Raw value get by key
func (v *Validation) Raw(key string) (any, bool) {
	if v.data == nil { // check input data
		return nil, false
	}
	return v.data.Get(key)
}

// RawVal value get by key
func (v *Validation) RawVal(key string) any {
	if v.data == nil { // check input data
		return nil
	}
	val, _ := v.data.Get(key)
	return val
}

// try to get value by key.
//
// **NOTE:**
//
// If v.data is StructData, will return zero value check. Other dataSource will always return `zero=False`.
func (v *Validation) tryGet(key string) (val any, exist, zero bool) {
	if v.data == nil {
		return
	}

	// CheckErr(skipCollect): safeData/filteredData 不收集,改用 1 槽对同字段连续读
	// 去重;其它字段落源(源已写回默认/过滤值,故读到已解析值,见计划 §4)。
	if v.skipCollect {
		if v.scKey == key {
			return v.scVal, true, false
		}
		return v.data.TryGet(key)
	}

	// find from filtered data.
	if val1, ok := v.filteredData[key]; ok {
		return val1, true, false
	}

	// find from validated data. (such as has default value)
	if val2, ok := v.safeData[key]; ok {
		return val2, true, false
	}

	// TODO add cache data v.caches[key]
	// get from source data
	return v.data.TryGet(key)
}

// Get value by key.
func (v *Validation) Get(key string) (val any, exist bool) {
	val, exist, _ = v.tryGet(key)
	return
}

// GetWithDefault get field value by key.
//
// On not found, if it has default value, will return default-value.
func (v *Validation) GetWithDefault(key string) (val any, exist, isDefault bool) {
	var zero bool
	val, exist, zero = v.tryGet(key)
	if exist && !zero {
		return
	}

	// try read custom default value
	defVal, isDefault := v.defValues[key]
	if isDefault {
		val = defVal
	}
	return
}

// Set value by key
func (v *Validation) Set(field string, val any) error {
	// check input data
	if v.data == nil {
		return ErrEmptyData
	}

	_, err := v.data.Set(field, val)
	return err
}

// only update set value by key for struct
func (v *Validation) updateValue(field string, val any) (any, error) {
	// data source is struct
	if v.data.Type() == sourceStruct {
		return v.data.Set(strings.TrimSuffix(field, ".*"), val)
	}

	// TODO dont update value for Form and Map data source
	return val, nil
}

// SetDefValue set a default value of given field
func (v *Validation) SetDefValue(field string, val any) {
	if v.defValues == nil {
		v.defValues = make(map[string]any)
	}
	v.defValues[field] = val
}

// GetDefValue get default value of the field
func (v *Validation) GetDefValue(field string) (any, bool) {
	defVal, ok := v.defValues[field]
	return defVal, ok
}

// SceneFields field names get
func (v *Validation) SceneFields() []string {
	return v.scenes[v.scene]
}

// scene field name map build. also (re)builds v.sceneWildcards for ".*" entries.
func (v *Validation) sceneFieldMap() (m map[string]uint8) {
	v.sceneWildcards = nil
	if v.scene == "" {
		return
	}

	if fields, ok := v.scenes[v.scene]; ok {
		// keep the map non-nil even when every field is skipped: a defined scene
		// that yields no fields (eg: scenes{"None": {""}}) must be distinguishable
		// from "no scene set" (nil map) in isNotNeedToCheck().
		m = make(map[string]uint8, len(fields))
		for _, field := range fields {
			// skip empty scene field. otherwise the "" key would match the empty
			// prefix fields[0:0] for every field and force-check everything (#314).
			if field == "" {
				continue
			}
			// ".*" wildcard entry (eg "Tags.*.Id"): kept apart so isNotNeedToCheck
			// can match it against indexed slice-element rule names like "Tags.0.Id"
			// (the scene field list otherwise matches by exact string only). (#283)
			if strings.Contains(field, ".*") {
				if v.sceneWildcards == nil {
					v.sceneWildcards = make(map[string]uint8)
				}
				v.sceneWildcards[field] = 1
				continue
			}
			m[field] = 1
		}
	}
	return
}

// Scene name get for current validation
func (v *Validation) Scene() string { return v.scene }

// IsOK for the validating
func (v *Validation) IsOK() bool { return !v.hasError }

// IsFail for the validating
func (v *Validation) IsFail() bool { return v.hasError }

// IsSuccess for the validating
func (v *Validation) IsSuccess() bool { return !v.hasError }

/*************************************************************
 * helper methods
 *************************************************************/

// on stop on error
func (v *Validation) shouldStop() bool {
	return v.hasError && v.StopOnError
}

// check current field is in optional parent field.
//
// return: true - optional parent field value is empty.
func (v *Validation) isInOptional(field string) bool {
	for name, flag := range v.optionals {
		// check like: field="Parent.Child" name="Parent"
		if strings.HasPrefix(field, name+".") {
			if flag != 0 {
				return flag == 1 // 1=empty
			}

			pVal, exist, zero := v.tryGet(name)
			if !exist || zero {
				v.optionals[name] = 1
				return true // not check field.
			}
			if IsEmpty(pVal) {
				v.optionals[name] = 1
				return true // not check field.
			}

			v.optionals[name] = 2
			return false
		}
	}

	return false
}

func (v *Validation) isNotNeedToCheck(field string) bool {
	// nil sceneFields AND no wildcard entries: no scene set (or scene not defined)
	// -> check all fields.
	if v.sceneFields == nil && len(v.sceneWildcards) == 0 {
		return false
	}

	// exact / ancestor-prefix match against the plain scene field list.
	// start at i=1: fields[0:0] is the empty prefix and never a valid scene key.
	if len(v.sceneFields) > 0 {
		fields := strings.Split(field, ".")
		for i := 1; i < len(fields); i++ {
			if _, ok := v.sceneFields[strings.Join(fields[0:i], ".")]; ok {
				return false
			}
		}
		if _, ok := v.sceneFields[field]; ok {
			return false
		}
	}

	// wildcard match: normalize numeric index segments to "*" and look up.
	// eg field "Tags.0.Id" -> "Tags.*.Id" matches scene entry "Tags.*.Id". (#283)
	if len(v.sceneWildcards) > 0 {
		if pat, hasIdx := indexPathToWildcard(field); hasIdx {
			if _, ok := v.sceneWildcards[pat]; ok {
				return false
			}
		}
	}

	return true
}
