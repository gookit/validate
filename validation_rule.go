package validate

import "strings"

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
