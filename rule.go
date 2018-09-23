package validate

import (
	"strings"
)

// Rules definition
type Rules []*Rule

// some global vars
var (
	rulesCaches map[string]Rules
)

/*************************************************************
 * validation rule
 *************************************************************/

// Rule definition
type Rule struct {
	// eg "create" "update"
	scene string
	// need validate fields.
	fields string
	// is optional, only validate on value is not empty.
	optional bool
	// default value setting
	defValue interface{}
	// error message(s)
	message  string
	messages map[string]string
	// want used filter name. allow multi filters. eg. "trim|int"
	filter  string
	filters map[string]int
	// validator name, allow multi validators. eg "min", "range", "required|min:2"
	validator string
	// arguments for the validator
	arguments []interface{}
	// some functions
	beforeFunc func(field string, v *Validation) bool // func (val interface{}) bool
	filterFunc interface{}                            // func (val interface{}) (newVal interface{})
	checkFunc  interface{}                            // func (val interface{}, ...) bool
	// custom check is empty.
	emptyChecker func(val interface{}) bool
}

// NewRule instance
func NewRule(fields, validator string, args ...interface{}) *Rule {
	return &Rule{
		fields:  fields,
		filters: make(map[string]int),
		// args
		arguments: args,
		validator: validator,
	}
}

// Setting the rule
func (r *Rule) Setting(fn func(r *Rule)) *Rule {
	fn(r)
	return r
}

// SetScene name for the rule.
func (r *Rule) SetScene(scene string) *Rule {
	r.scene = scene
	return r
}

// SetCheckFunc use custom check func.
func (r *Rule) SetCheckFunc(checkFunc interface{}) *Rule {
	r.checkFunc = checkFunc
	return r
}

// SetOptional only validate on value is not empty.
func (r *Rule) SetOptional(optional bool) *Rule {
	r.optional = optional
	return r
}

// SetMessage set error message
func (r *Rule) SetMessage(errMsg string) *Rule {
	r.message = errMsg
	return r
}

// SetMessages set error message map
func (r *Rule) SetMessages(msgMap SMap) *Rule {
	r.messages = msgMap
	return r
}

// UseFilters add filter name(s)
func (r *Rule) UseFilters(names ...string) *Rule {
	for _, name := range names {
		r.filters[name] = 1
	}

	return r
}

// FilterWithArgs
func (r *Rule) FilterWithArgs(name string, args ...interface{}) *Rule {
	// r.filterFunc = msgMap
	return r
}

// Fields names list
func (r *Rule) Fields() []string {
	return stringSplit(r.fields, ",")
}

// Apply rule for the rule fields
func (r *Rule) Apply(v *Validation) bool {
	fieldMap := v.SceneFieldMap()
	dontNeedCheck := func(field string) bool {
		if len(fieldMap) == 0 {
			return false
		}

		_, ok := fieldMap[field]
		return ok
	}

	// validate field
	for _, field := range r.Fields() {
		if dontNeedCheck(field) {
			continue
		}

		// only one validator
		if !strings.ContainsRune(r.validator, ',') {
			r.Validate(field, r.validator, v)
		} else { // has multi validators
			vs := stringSplit(r.validator, "|")

			for _, validator := range vs {
				// stop on error
				if r.Validate(field, validator, v) && v.StopOnError {
					return true
				}
			}
		}

		// stop on error
		if v.shouldStop() {
			return true
		}
	}

	return false
}
