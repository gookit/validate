package validate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const errorName = "validate"
const defaultTag = "validate"
const defaultMaxMemory int64 = 32 << 20 // 32 MB

// SMap is short name for map[string]string
type SMap map[string]string

// SValues simple values
type SValues map[string][]string

// GMap is short name for map[string]interface{}
type GMap map[string]interface{}

// DataFace interface definition
type DataFace interface {
	Get(key string) (interface{}, bool)
	Set(field string, val interface{}) error
	// validation instance create func
	Create(scene ...string) *Validation
	Validation(scene ...string) *Validation
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
	TagName: defaultTag,
}

// GlobalOption settings for validate
type GlobalOption struct {
	// TagName in the struct tags.
	TagName string
	// StopOnError If true: An error occurs, it will cease to continue to verify
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty
	SkipOnEmpty bool
}

// Validation definition
type Validation struct {
	DataFace
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
	// all validated fields list
	fields []string
	// mark has error occurs
	hasError bool
	// mark is validated
	validated bool
	// translator
	trans *Translator
	// rules for the validation
	rules []*Rule
	// current scene name
	scene string
	// scenes config.
	// {
	// 	"create": {"field", "field1"}
	// 	"update": {"field", "field2"}
	// }
	scenes SValues
	// filters and functions for the validation
	filterFuncs GMap
	// filter func reflect.Value map
	filterValues map[string]reflect.Value
	// validators and functions for the validation
	validators GMap
	// validator func reflect.Value map
	validatorValues map[string]reflect.Value
}

// NewValidation instance
func NewValidation(data DataFace, scene ...string) *Validation {
	v := &Validation{
		Errors: make(Errors),
		// add data source
		DataFace: data,
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

	return v.SetScene(scene...)
}

func newWithError(d DataFace, err error) *Validation {
	return d.Create().WithError(err)
}

/*************************************************************
 * validation settings
 *************************************************************/

// Config the Validation instance
func (v *Validation) Config(fn func(v *Validation)) {
	fn(v)
}

// Reset the Validation instance
func (v *Validation) Reset() {
	// v.trans = NewTranslator()
	v.rules = v.rules[:0]
	v.Errors = Errors{}
	v.hasError = false
	v.validated = false
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

// SetRulesFromCaches
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
// 	v.StringRule("name", "required|string|min:12")
func (v *Validation) StringRule(field, ruleString string) *Validation {
	ruleString = strings.TrimSpace(ruleString)
	ruleString = strings.Trim(ruleString, "|:")

	rules := stringSplit(ruleString, "|")
	for _, validator := range rules {
		validator = strings.Trim(validator, ":")
		if validator == "" { // empty
			continue
		}

		// has args
		if strings.ContainsRune(validator, ':') {
			list := stringSplit(validator, ":")
			v.AddRule(field, list[0], strings2Args(list[1:])...)
		} else {
			v.AddRule(field, validator)
		}
	}

	return v
}

// StringRules add multi rules by string map
// Usage:
// 	v.StringRules(map[string]string{
// 		"name": "required|string|min:12",
// 		"age": "required|int|min:12",
// 	})
func (v *Validation) StringRules(sMap SMap) *Validation {
	for name, rule := range sMap {
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
		// checkFunc: v.GetValidator(validator),
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
 * do Validate
 *************************************************************/

// Validate processing
func (v *Validation) Validate(scene ...string) bool {
	// has been validated OR has error
	if v.validated || v.shouldStop() {
		return v.IsSuccess()
	}

	v.SetScene(scene...)

	// apply rule to validate data.
	for _, rule := range v.rules {
		// has error and v.StopOnError is true.
		if rule.Apply(v) {
			break
		}
	}

	v.validated = true
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

// Messages settings. you can custom validator error messages.
// Usage:
// 	v.WithMessages(map[string]string{
// 		"require": "oh! {field} is required",
// 		"range": "oh! {field} must be in the range %d - %d",
//  })
func (v *Validation) WithMessages(m map[string]string) *Validation {
	v.trans.LoadMessages(m)
	return v
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

/*************************************************************
 * getter methods
 *************************************************************/

// Get value by key
func (v *Validation) Get(key string) (interface{}, bool) {
	if v.DataFace == nil { // check input data
		return nil, false
	}

	return v.DataFace.Get(key)
}

// Set value by key
func (v *Validation) Set(field string, val interface{}) error {
	if v.DataFace == nil { // check input data
		return ErrEmptyData
	}

	return v.DataFace.Set(field, val)
}

// Trans get message Translator
func (v *Validation) Trans() *Translator {
	return v.trans
}

// SceneFields field names get
func (v *Validation) SceneFields() []string {
	return v.scenes[v.scene]
}

// SceneFieldMap field name map build and get
func (v *Validation) SceneFieldMap() (m map[string]uint8) {
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

/*************************************************************
 * helper methods
 *************************************************************/

func (v *Validation) shouldStop() bool {
	return v.hasError && v.StopOnError
}

// Safe
func (v *Validation) Safe(field string) {

}

func panicf(format string, args ...interface{}) {
	panic("validate: " + fmt.Sprintf(format, args...))
}
