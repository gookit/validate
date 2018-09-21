package validate

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

const defaultMaxMemory int64 = 32 << 20 // 32 MB

// SMap is short name for map[string]string
type SMap map[string]string

// SValues simple values
type SValues map[string][]string

// GMap is short name for map[string]interface{}
type GMap map[string]interface{}

// DataFace interface definition
type DataFace interface {
	// Int() int
	// Int64() int
	Get(key string) (interface{}, bool)
	Set(field string, val interface{}) error
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

// ErrInvalidType
type ErrInvalidType struct {
	Type reflect.Type
}

func (e *ErrInvalidType) Error() string {
	if e.Type == nil {
		return "validator: (nil)"
	}

	return "validator: (nil " + e.Type.String() + ")"
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

/*************************************************************
 * validation settings
 *************************************************************/

// NewValidation instance
func NewValidation(d DataFace, scene ...string) *Validation {
	v := &Validation{
		Errors:   make(Errors),
		DataFace: d,
		// default config
		StopOnError: true,
		SkipOnEmpty: true,
		// create message translator
		trans: NewTranslator(),
	}

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

// WithScenarios is alias of the WithScenes()
func (v *Validation) WithScenarios(scenes SValues) *Validation {
	return v.WithScenes(scenes)
}

// WithScenes config.
// Usage:
//	v.WithScenes(SValues{
// 		"create": []string{"name", "email"},
// 		"update": []string{"name"},
// 	})
//	ok := v.AtScene("create").Validate()
func (v *Validation) WithScenes(scenes map[string][]string) *Validation {
	v.scenes = scenes
	return v
}

// SetRules
func (v *Validation) SetRules(rules ...*Rule) *Validation {
	v.rules = rules
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

// AppendRule
func (v *Validation) AppendRule(rule *Rule) *Rule {
	v.rules = append(v.rules, rule)
	return rule
}

// StringRule add field rules by string
// Usage:
// 	v.StringRule("name", "required|string|min:12")
func (v *Validation) StringRule(field, ruleString string) *Validation {
	ruleString = strings.TrimSpace(ruleString)
	ruleString = strings.Trim(ruleString, "|:")

	rules := stringSplit(ruleString, "|")
	for _, singleRule := range rules {
		singleRule = strings.Trim(singleRule, ":")
		if singleRule == "" { // empty
			continue
		}

		if strings.ContainsRune(singleRule, ':') { // has args
			list := stringSplit(singleRule, ":")
			v.AddRule(field, list[0], strings2Args(list[1:]))
		} else {
			v.AddRule(field, singleRule)
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

// AddRule
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
		return v.AtScene(scene[0])
	}

	return v
}

func (v *Validation) Custom(fields, validator string, args ...interface{}) *Rule {
	return v.AddRule(fields, validator, args...)
}

/*************************************************************
 * do Validate
 *************************************************************/

// Validate processing
func (v *Validation) Validate(scene ...string) bool {
	if v.validated {
		return v.IsSuccess()
	}

	v.SetScene(scene...)

	// apply rule to validate data.
	for _, rule := range v.rules {
		if rule.Apply(v) {
			break
		}
	}

	v.validated = true
	return v.IsSuccess()
}

func (v *Validation) shouldStop() bool {
	return v.hasError && v.StopOnError
}

/*************************************************************
 * errors messages
 *************************************************************/

// WithTranslates set fields translates. Usage:
// 	v.WithTranslates(map[string]string{
//		"name": "User Name",
//		"pwd": "Password",
//  })
func (v *Validation) WithTranslates(m map[string]string) *Validation {
	v.trans.SetFieldMap(m)
	return v
}

// Messages settings. Usage:
// 	v.WithMessages(map[string]string{
//		"name": "User Name",
//		"pwd": "Password",
//  })
func (v *Validation) WithMessages(m map[string]string) *Validation {
	v.trans.Load(m)
	return v
}

// Fields returns the fields for all validated
func (v *Validation) Fields() []string {
	return v.fields
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

// SceneFields name get
func (v *Validation) SceneFields() (fields []string) {
	if v.scene == "" {
		return
	}

	return v.scenes[v.scene]
}

// SceneFieldMap name get
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

// Scene name get
func (v *Validation) Scene() string {
	return v.scene
}

// IsOK for the validate
func (v *Validation) IsOK() bool {
	return !v.hasError
}

func (v *Validation) IsFail() bool {
	return v.hasError
}

func (v *Validation) IsSuccess() bool {
	return !v.hasError
}

/*************************************************************
 * helper methods
 *************************************************************/

// shouldCheck field
func (v *Validation) shouldCheck(field string) bool {
	return false
}

// Safe
func (v *Validation) Safe(field string) {

}

// Reset
func (v *Validation) Reset() {
	v.trans = NewTranslator()
	v.rules = v.rules[:0]
	v.Errors = Errors{}
	v.hasError = false
}

func panicf(format string, args ...interface{}) {
	panic("validate: " + fmt.Sprintf(format, args...))
}
