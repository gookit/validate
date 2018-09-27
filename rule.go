package validate

import (
	"reflect"
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
	// need validate fields. allow multi. eg "field1, field2"
	fields string
	// is optional, only validate on value is not empty.
	optional bool
	// default value setting
	defValue interface{}
	// error message(s)
	message  string
	messages map[string]string
	// filter map. can with args. eg. "int", "str2arr:,"
	filters map[string]string
	// validator name, allow multi validators. eg "min", "range", "required"
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
		fields: fields,
		// filters
		filters: make(map[string]string),
		// validator args
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

// UseFilters add filter(s)
func (r *Rule) UseFilters(filters ...string) *Rule {
	for _, filter := range filters {
		pos := strings.IndexRune(filter, ':')

		// has args
		if pos > 0 {
			name := filter[:pos]
			r.filters[name] = filter[pos+1:]
		} else {
			r.filters[filter] = ""
		}
	}

	return r
}

// Fields names list
func (r *Rule) Fields() []string {
	return stringSplit(r.fields, ",")
}

// Apply rule for the rule fields
func (r *Rule) Apply(v *Validation) (stop bool) {
	fieldMap := v.SceneFieldMap()
	dontNeedCheck := func(field string) bool {
		if len(fieldMap) == 0 {
			return false
		}

		_, ok := fieldMap[field]
		return ok
	}

	// validate field value
	for _, field := range r.Fields() {
		if dontNeedCheck(field) {
			continue
		}

		val, has := v.Get(field) // get field value.
		if !has && v.StopOnError { // no field AND stop on error
			return true
		}

		// apply filters func
		val, err := applyFilters(val, r.filters, v)
		if err != nil { // has error
			v.AddError(filterError, err.Error())
			return true
		} else { // save filtered value.
			v.filteredData[field] = val
		}

		// only one validator
		if !strings.ContainsRune(r.validator, '|') {
			r.Validate(field, r.validator, val, v)
		} else { // has multi validators
			vs := stringSplit(r.validator, "|")
			for _, validator := range vs {
				// stop on error
				if r.Validate(field, validator, val, v) && v.StopOnError {
					return true
				}
			}
		}

		// stop on error
		if v.shouldStop() {
			return true
		}

		// save validated value.
		v.safeData[field] = val
	}

	return false
}

/*************************************************************
 * filtering rule
 *************************************************************/

// FilterRule definition
type FilterRule struct {
	// fields to filter
	fields []string
	// filter list, can with args. eg. "int" "str2arr:,"
	filters map[string]string
}

func newFilterRule(fields []string) *FilterRule {
	return &FilterRule{
		fields:  fields,
		filters: make(map[string]string),
	}
}

// UseFilters add filter(s)
func (r *FilterRule) UseFilters(filters ...string) *FilterRule {
	return r.AddFilters(filters...)
}

// AddFilters add filter(s).
// Usage:
// 	r.AddFilters("int", "str2arr:,")
func (r *FilterRule) AddFilters(filters ...string) *FilterRule {
	for _, filter := range filters {
		pos := strings.IndexRune(filter, ':')

		// has filter args
		if pos > 0 {
			name := filter[:pos]
			r.filters[name] = filter[pos+1:]
		} else {
			r.filters[filter] = ""
		}
	}

	return r
}

// Apply rule for the rule fields
func (r *FilterRule) Apply(v *Validation) (err error) {
	// validate field
	for _, field := range r.Fields() {
		// get field value.
		val, has := v.Get(field)
		if !has { // no field
			continue
		}

		// call filters
		val, err = applyFilters(val, r.filters, v)
		if err != nil {
			v.AddError(filterError, err.Error())
			return err
		}

		// save filtered value.
		v.filteredData[field] = val
		// v.safeData[field] = val
	}

	return
}

// Fields name get
func (r *FilterRule) Fields() []string {
	return r.fields
}

func applyFilters(val interface{}, filters map[string]string, v *Validation) (interface{}, error) {
	var err error

	// call filters
	for name, argStr := range filters {
		fv := v.FilterFuncValue(name)
		args := parseArgString(argStr)

		val, err = callFilter(fv, val, strings2Args(args))
		if err != nil {
			return nil, err
		}
	}

	return val, nil
}

func parseArgString(argStr string) (ss []string) {
	if argStr == "" { // no arg
		return
	}

	if len(argStr) == 1 { // one char
		return []string{argStr}
	}

	return stringSplit(argStr, ",")
}

func callFilter(fv reflect.Value, val interface{}, args []interface{}) (interface{}, error) {
	var rs []reflect.Value
	if len(args) > 0 {
		rs = CallByValue(fv, buildArgs(val, args)...)
	} else {
		rs = CallByValue(fv, val)
	}

	rl := len(rs)

	// return new val.
	if rl > 0 {
		val = rs[0].Interface()

		if rl == 2 { // filter func report error
			if err := rs[1].Interface(); err != nil {
				return nil, err.(error)
			}
		}
	}

	return val, nil
}
