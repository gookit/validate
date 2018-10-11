package validate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const errorName = "_validate"
const filterError = "_filter"
const defaultTag = "validate"
const defaultFilterTag = "filter"
const defaultMaxMemory int64 = 32 << 20 // 32 MB

// MS is short name for map[string]string
type MS map[string]string

// SValues simple values
type SValues map[string][]string

// M is short name for map[string]interface{}
type M map[string]interface{}

// DataFace interface definition
type DataFace interface {
	Get(key string) (interface{}, bool)
	Set(field string, val interface{}) error
	// validation instance create func
	Create(err ...error) *Validation
	Validation(err ...error) *Validation
}

// MarshalFunc define
type MarshalFunc func(v interface{}) ([]byte, error)

// UnmarshalFunc define
type UnmarshalFunc func(data []byte, v interface{}) error

// data (Un)marshal func
var (
	Marshal   MarshalFunc   = json.Marshal
	Unmarshal UnmarshalFunc = json.Unmarshal
)

var timeType = reflect.TypeOf(time.Time{})
var globalOpt = &GlobalOption{
	StopOnError: true,
	SkipOnEmpty: true,
	// tag name in struct tags
	FilterTag: defaultFilterTag,
	// tag name in struct tags
	ValidateTag: defaultTag,
}

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
}

// Validation definition
type Validation struct {
	// source input data
	data DataFace
	// all validated fields list
	fields []string
	// filtered clean data
	filteredData M
	// filtered/validated safe data
	safeData M
	// Errors for the validate
	Errors Errors
	// CacheKey for cache rules
	CacheKey string
	// StopOnError If true: An error occurs, it will cease to continue to verify
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty
	SkipOnEmpty bool
	// CachingRules switch. default is False
	CachingRules bool
	// mark has error occurs
	hasError bool
	// mark is filtered
	filtered bool
	// mark is validated
	validated bool
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
	// 	"create": {"field", "field1"}
	// 	"update": {"field", "field2"}
	// }
	scenes SValues
	// should checked fields in current scene.
	sceneFields map[string]uint8
	// filtering rules for the validation
	filterRules []*FilterRule
	// filters and functions for the validation
	filterFuncs M
	// filter func reflect.Value map
	filterValues map[string]reflect.Value
}

// NewValidation instance
func NewValidation(data DataFace, scene ...string) *Validation {
	v := &Validation{
		Errors: make(Errors),
		// add data source
		data: data,
		// validated data
		safeData:   make(map[string]interface{}),
		validators: make(map[string]int),
		// filtered data
		filteredData: make(map[string]interface{}),
		// default config
		StopOnError: globalOpt.StopOnError,
		SkipOnEmpty: globalOpt.SkipOnEmpty,
		// create message translator
		trans: NewTranslator(),
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
func (v *Validation) Config(fn func(v *Validation)) {
	fn(v)
}

// ResetResult reset the validate result.
func (v *Validation) ResetResult() {
	v.Errors = Errors{}
	v.hasError = false
	v.filtered = false
	v.validated = false
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
}

// WithScenarios is alias of the WithScenes()
func (v *Validation) WithScenarios(scenes SValues) *Validation {
	return v.WithScenes(scenes)
}

// WithScenes config.
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

// SetRulesFromCaches key string
func (v *Validation) SetRulesFromCaches(key string) *Validation {
	v.rules = rulesCaches[key]
	return v
}

// CacheRules to caches for repeat use.
func (v *Validation) CacheRules(key string) {
	if rulesCaches == nil {
		rulesCaches = make(map[string]Rules)
	}

	rulesCaches[key] = v.rules
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
 * add validate rules
 *************************************************************/

// StringRule add field rules by string
// Usage:
// 	v.StringRule("name", "required|string|minLen:6")
// 	// will try convert to int before apply validate.
// 	v.StringRule("age", "required|int|min:12", "toInt")
func (v *Validation) StringRule(field, rule string, filterRule ...string) *Validation {
	rule = strings.TrimSpace(rule)
	rules := stringSplit(strings.Trim(rule, "|:"), "|")
	for _, validator := range rules {
		validator = strings.Trim(validator, ":")
		if validator == "" { // empty
			continue
		}

		// has args
		if strings.ContainsRune(validator, ':') {
			list := stringSplit(validator, ":")
			args := parseArgString(list[1])
			name := ValidatorName(list[0])
			switch name {
			// some special validator. need merge args to one.
			case "enum", "notIn":
				v.AddRule(field, list[0], args)
			default:
				v.AddRule(field, list[0], strings2Args(args)...)
			}
		} else {
			v.AddRule(field, validator)
		}
	}

	if len(filterRule) > 0 {
		v.FilterRule(field, filterRule[0])
	}

	return v
}

// StringRules add multi rules by string map
// Usage:
// 	v.StringRules(map[string]string{
// 		"name": "required|string|min:12",
// 		"age": "required|int|min:12",
// 	})
func (v *Validation) StringRules(mp MS) *Validation {
	for name, rule := range mp {
		v.StringRule(name, rule)
	}

	return v
}

// AddRule for current validate
func (v *Validation) AddRule(fields, validator string, args ...interface{}) *Rule {
	rule := &Rule{
		fields: fields,
		// args for the validator
		arguments: args,
		validator: validator,
	}

	v.rules = append(v.rules, rule)
	return rule
}

// AppendRule instance
func (v *Validation) AppendRule(rule *Rule) *Rule {
	v.rules = append(v.rules, rule)
	return rule
}

/*************************************************************
 * Do Validate
 *************************************************************/

// Validate processing
func (v *Validation) Validate(scene ...string) bool {
	// has been validated OR has error
	if v.validated || v.shouldStop() {
		return v.IsSuccess()
	}

	v.SetScene(scene...)
	v.sceneFields = v.sceneFieldMap()

	// apply filter rule before validate.
	if !v.Filtering() {
		return v.IsSuccess()
	}

	// apply rule to validate data.
	for _, rule := range v.rules {
		// has error and v.StopOnError is true.
		if rule.Apply(v) {
			break
		}
	}

	v.validated = true
	if v.hasError { // clear safe data.
		v.safeData = make(map[string]interface{})
	}

	return v.IsSuccess()
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
	if v.filtered {
		return v.IsSuccess()
	}

	// apply rule to validate data.
	for _, rule := range v.filterRules {
		if err := rule.Apply(v); err != nil { // has error
			v.AddError(filterError, err.Error())
			break
		}
	}

	v.filtered = true
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

// Fields returns the fields for all validated
func (v *Validation) Fields() []string {
	return v.fields
}

// WithError add error of the validation
func (v *Validation) WithError(err error) *Validation {
	if err != nil {
		v.AddError(errorName, err.Error())
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

	return v.data.Get(key)
}

// Safe get safe value by key
func (v *Validation) Safe(key string) (val interface{}, ok bool) {
	if v.data == nil { // check input data
		return
	}

	val, ok = v.safeData[key]
	return
}

// Set value by key
func (v *Validation) Set(field string, val interface{}) error {
	if v.data == nil { // check input data
		return ErrEmptyData
	}

	return v.data.Set(field, val)
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

// FilteredData get
func (v *Validation) FilteredData() M {
	return v.filteredData
}

/*************************************************************
 * helper methods
 *************************************************************/

func (v *Validation) shouldStop() bool {
	return v.hasError && v.StopOnError
}

func (v *Validation) isNoNeedToCheck(field string) bool {
	if len(v.sceneFields) == 0 {
		return false
	}

	_, ok := v.sceneFields[field]
	return !ok
}
