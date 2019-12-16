package validate

import (
	"fmt"
	"reflect"
)

// some default value settings.
const (
	filterTag     = "filter"
	filterError   = "_filter"
	validateTag   = "validate"
	validateError = "_validate"
	// sniff Length, use for detect file mime type
	sniffLen = 512
	// 32 MB
	defaultMaxMemory int64 = 32 << 20
)

// M is short name for map[string]interface{}
type M map[string]interface{}

// MS is short name for map[string]string
type MS map[string]string

// SValues simple values
type SValues map[string][]string

// GlobalOption settings for validate
type GlobalOption struct {
	// FilterTag name in the struct tags.
	FilterTag string
	// ValidateTag in the struct tags.
	ValidateTag string
	// StopOnError If true: An error occurs, it will cease to continue to verify
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty
	SkipOnEmpty bool
	// UpdateSource Whether to update source field value, useful for struct validate
	UpdateSource bool
	// CheckDefault Whether to validate the default value set by the user
	CheckDefault bool
	// CheckZero Whether validate the default zero value. (intX,uintX: 0, string: "")
	CheckZero bool
}

var globalOpt = &GlobalOption{
	StopOnError: true,
	SkipOnEmpty: true,
	// tag name in struct tags
	FilterTag: filterTag,
	// tag name in struct tags
	ValidateTag: validateTag,
}

// Validation definition
type Validation struct {
	// source input data
	data DataFace
	// all validated fields list
	// fields []string
	// filtered/validated safe data
	safeData M
	// filtered clean data
	filteredData M
	// Errors for the validate
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
	// CachingRules switch. default is False
	// CachingRules bool
	// save user set default values
	defValues map[string]interface{}
	// mark has error occurs
	hasError bool
	// mark is filtered
	hasFiltered bool
	// mark is validated
	hasValidated bool
	// validate rules for the validation
	rules []*Rule
	// validators for the validation
	validators map[string]int
	// validator func meta info
	validatorMetas map[string]*funcMeta
	// validator func reflect.Value map
	validatorValues map[string]reflect.Value
	// translator instance
	trans *Translator
	// current scene name
	scene string
	// scenes config.
	// {
	// 	"create": {"field0", "field1"}
	// 	"update": {"field0", "field2"}
	// }
	scenes SValues
	// should checked fields in current scene.
	sceneFields map[string]uint8
	// filtering rules for the validation
	filterRules []*FilterRule
	// filter func reflect.Value map
	filterValues map[string]reflect.Value
}

// NewEmpty new validation instance, but not add data.
func NewEmpty(scene ...string) *Validation {
	return NewValidation(nil, scene...)
}

// NewValidation new validation instance
func NewValidation(data DataFace, scene ...string) *Validation {
	v := &Validation{
		Errors: make(Errors),
		// add data source
		data: data,
		// create message translator
		trans: NewTranslator(),
		// validated data
		safeData: make(map[string]interface{}),
		// validator names
		validators: make(map[string]int),
		// filtered data
		filteredData: make(map[string]interface{}),
		// default config
		StopOnError: globalOpt.StopOnError,
		SkipOnEmpty: globalOpt.SkipOnEmpty,
	}

	// init build in context validator
	v.validatorValues = map[string]reflect.Value{
		"required": reflect.ValueOf(v.Required),
		// field compare
		"eqField":  reflect.ValueOf(v.EqField),
		"neField":  reflect.ValueOf(v.NeField),
		"gtField":  reflect.ValueOf(v.GtField),
		"gteField": reflect.ValueOf(v.GteField),
		"ltField":  reflect.ValueOf(v.LtField),
		"lteField": reflect.ValueOf(v.LteField),
		// file upload check
		"isFile":      reflect.ValueOf(v.IsFile),
		"isImage":     reflect.ValueOf(v.IsImage),
		"inMimeTypes": reflect.ValueOf(v.InMimeTypes),
	}

	v.validatorMetas = make(map[string]*funcMeta)

	// collect meta info
	for n, fv := range v.validatorValues {
		v.validators[n] = 1 // built in
		v.validatorMetas[n] = newFuncMeta(n, true, fv)
	}

	return v.SetScene(scene...)
}

func newWithError(d DataFace, err error) *Validation {
	if d == nil {
		if err != nil {
			return NewValidation(d).WithError(err)
		}
		return NewValidation(d)
	}

	return d.Validation(err)
}

/*************************************************************
 * validation settings
 *************************************************************/

// Config the Validation instance
// func (v *Validation) Config(fn func(v *Validation)) {
// 	fn(v)
// }

// ResetResult reset the validate result.
func (v *Validation) ResetResult() {
	v.Errors = Errors{}
	v.hasError = false
	v.hasFiltered = false
	v.hasValidated = false
	// result data
	v.safeData = make(map[string]interface{})
	v.filteredData = make(map[string]interface{})
}

// Reset the Validation instance
func (v *Validation) Reset() {
	v.ResetResult()

	// rules
	v.rules = v.rules[:0]
	v.filterRules = v.filterRules[:0]
	v.validators = make(map[string]int)
}

// WithScenarios is alias of the WithScenes()
func (v *Validation) WithScenarios(scenes SValues) *Validation {
	return v.WithScenes(scenes)
}

// WithScenes set scene config.
// Usage:
// 	v.WithScenes(SValues{
// 		"create": []string{"name", "email"},
// 		"update": []string{"name"},
// 	})
// 	ok := v.AtScene("create").Validate()
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

// AddValidators to the Validation
func (v *Validation) AddValidators(m map[string]interface{}) {
	for name, checkFunc := range m {
		v.AddValidator(name, checkFunc)
	}
}

// AddValidator to the Validation. checkFunc must return a bool
func (v *Validation) AddValidator(name string, checkFunc interface{}) {
	fv := checkValidatorFunc(name, checkFunc)

	v.validators[name] = 2 // custom
	v.validatorValues[name] = fv
	v.validatorMetas[name] = newFuncMeta(name, false, fv)
}

// ValidatorMeta get by name
func (v *Validation) validatorMeta(name string) *funcMeta {
	// current validation
	if fm, ok := v.validatorMetas[name]; ok {
		return fm
	}

	// from global validators
	if fm, ok := validatorMetas[name]; ok {
		return fm
	}

	// if v.data is StructData instance.
	if sd, ok := v.data.(*StructData); ok {
		fv, ok := sd.FuncValue(name)
		if ok {
			fm := newFuncMeta(name, false, fv)
			// storage it.
			v.validators[name] = 2 // custom
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

	// global validators
	_, ok := validatorMetas[name]
	return ok
}

// Validators get all validator names
func (v *Validation) Validators(withGlobal bool) map[string]int {
	if withGlobal {
		mp := make(map[string]int)
		for name, typ := range validators {
			mp[name] = typ
		}

		for name, typ := range v.validators {
			mp[name] = typ
		}
		return mp
	}

	return v.validators
}

/*************************************************************
 * Do Validate
 *************************************************************/

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
	if false == v.Filtering() && v.StopOnError {
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

// ValidateData validate given data
func (v *Validation) ValidateData(data DataFace) bool {
	v.data = data
	return v.Validate()
}

/*************************************************************
 * Do filtering/sanitize
 *************************************************************/

// Sanitize data by filter rules
func (v *Validation) Sanitize() bool {
	return v.Filtering()
}

// Filtering data by filter rules
func (v *Validation) Filtering() bool {
	if v.hasFiltered {
		return v.IsSuccess()
	}

	// apply rule to validate data.
	for _, rule := range v.filterRules {
		if err := rule.Apply(v); err != nil { // has error
			v.AddError(filterError, err.Error())
			break
		}
	}

	v.hasFiltered = true
	return v.IsSuccess()
}

/*************************************************************
 * errors messages
 *************************************************************/

// WithTranslates settings.you can custom field translates.
// Usage:
// 	v.WithTranslates(map[string]string{
// 		"name": "User Name",
// 		"pwd": "Password",
//  })
func (v *Validation) WithTranslates(m map[string]string) *Validation {
	v.trans.AddFieldMap(m)
	return v
}

// AddTranslates settings data. like WithTranslates()
func (v *Validation) AddTranslates(m map[string]string) {
	v.trans.AddFieldMap(m)
}

// WithMessages settings. you can custom validator error messages.
// Usage:
// 	v.WithMessages(map[string]string{
// 		"require": "oh! {field} is required",
// 		"range": "oh! {field} must be in the range %d - %d",
//  })
func (v *Validation) WithMessages(m map[string]string) *Validation {
	v.trans.LoadMessages(m)
	return v
}

// AddMessages settings data. like WithMessages()
func (v *Validation) AddMessages(m map[string]string) {
	v.trans.LoadMessages(m)
}

// WithError add error of the validation
func (v *Validation) WithError(err error) *Validation {
	if err != nil {
		v.AddError(validateError, err.Error())
	}

	return v
}

// AddError message for a field
func (v *Validation) AddError(field string, msg string) {
	if !v.hasError {
		v.hasError = true
	}

	v.Errors.Add(field, msg)
}

// AddErrorf add a formatted error message
func (v *Validation) AddErrorf(field, msgFormat string, args ...interface{}) {
	v.AddError(field, fmt.Sprintf(msgFormat, args...))
}

func (v *Validation) convertArgTypeError(name string, argKind, wantKind reflect.Kind) {
	v.AddErrorf("_convert", "cannot convert %s to %s, validator '%s'", argKind, wantKind, name)
}

/*************************************************************
 * getter methods
 *************************************************************/

// Raw value get by key
func (v *Validation) Raw(key string) (interface{}, bool) {
	if v.data == nil { // check input data
		return nil, false
	}

	return v.data.Get(key)
}

// Get value by key
func (v *Validation) Get(key string) (interface{}, bool) {
	if v.data == nil { // check input data
		return nil, false
	}

	// find from filtered data.
	if val, ok := v.filteredData[key]; ok {
		return val, true
	}

	// find from validated data. (such as has default value)
	if val, ok := v.safeData[key]; ok {
		return val, true
	}

	// get from source data
	return v.data.Get(key)
}

// Filtered get filtered value by key
func (v *Validation) Filtered(key string) interface{} {
	val, _ := v.filteredData[key]
	return val
}

// Safe get safe value by key
func (v *Validation) Safe(key string) (val interface{}, ok bool) {
	if v.data == nil { // check input data
		return
	}

	val, ok = v.safeData[key]
	return
}

// SafeVal get safe value by key
func (v *Validation) SafeVal(key string) interface{} {
	val, _ := v.Safe(key)
	return val
}

// GetSafe get safe value by key
func (v *Validation) GetSafe(key string) interface{} {
	val, _ := v.Safe(key)
	return val
}

// BindSafeData to a struct.
func (v *Validation) BindSafeData(ptr interface{}) error {
	if len(v.safeData) == 0 { // no safe data.
		return nil
	}

	// to json bytes
	bts, err := Marshal(v.safeData)
	if err != nil {
		return err
	}

	return Unmarshal(bts, ptr)
}

// Set value by key
func (v *Validation) Set(field string, val interface{}) error {
	// check input data
	if v.data == nil {
		return ErrEmptyData
	}

	_, err := v.data.Set(field, val)
	return err
}

// only update set value by key for struct
func (v *Validation) updateValue(field string, val interface{}) (interface{}, error) {
	// data source is struct
	// if _, ok := v.data.(*StructData); ok {
	if v.data.Type() == uint8(sourceStruct) {
		return v.data.Set(field, val)
	}

	// TODO dont update value for Form and Map data source
	return val, nil
}

// SetDefValue set an default value of given field
func (v *Validation) SetDefValue(field string, val interface{}) {
	if v.defValues == nil {
		v.defValues = make(map[string]interface{})
	}

	v.defValues[field] = val
}

// GetDefValue get default value of the field
func (v *Validation) GetDefValue(field string) (interface{}, bool) {
	defVal, ok := v.defValues[field]
	return defVal, ok
}

// Trans get message Translator
func (v *Validation) Trans() *Translator {
	return v.trans
}

// SceneFields field names get
func (v *Validation) SceneFields() []string {
	return v.scenes[v.scene]
}

// scene field name map build
func (v *Validation) sceneFieldMap() (m map[string]uint8) {
	if v.scene == "" {
		return
	}

	if fields, ok := v.scenes[v.scene]; ok {
		m = make(map[string]uint8, len(fields))
		for _, field := range fields {
			m[field] = 1
		}
	}
	return
}

// Scene name get for current validation
func (v *Validation) Scene() string {
	return v.scene
}

// IsOK for the validate
func (v *Validation) IsOK() bool {
	return !v.hasError
}

// IsFail for the validate
func (v *Validation) IsFail() bool {
	return v.hasError
}

// IsSuccess for the validate
func (v *Validation) IsSuccess() bool {
	return !v.hasError
}

// SafeData get all validated safe data
func (v *Validation) SafeData() M {
	return v.safeData
}

// FilteredData return filtered data.
func (v *Validation) FilteredData() M {
	return v.filteredData
}

/*************************************************************
 * helper methods
 *************************************************************/

func (v *Validation) shouldStop() bool {
	return v.hasError && v.StopOnError
}

func (v *Validation) isNotNeedToCheck(field string) bool {
	if len(v.sceneFields) == 0 {
		return false
	}

	_, ok := v.sceneFields[field]
	return !ok
}
