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
	// filter map. no args. eg. "int"
	filters map[string]int
	// filter map, with args. eg. "toArray:,"
	argFilters map[string][]interface{}
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
		// filter
		filters:    make(map[string]int),
		argFilters: make(map[string][]interface{}),
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

// UseFilters add filter name(s)
func (r *Rule) UseFilters(names ...string) *Rule {
	for _, name := range names {
		r.filters[name] = 1
	}

	return r
}

// UseArgsFilter add filter with args
func (r *Rule) UseArgsFilter(name string, args ...interface{}) *Rule {
	r.argFilters[name] = args
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

		// get field value.
		val, has := v.Get(field)
		if !has && v.StopOnError { // no field AND stop on error
			return true
		}

		// apply filters func
		val, goon := r.ApplyFilters(val, v)
		if !goon { // has error
			return true
		}

		// only one validator
		if !strings.ContainsRune(r.validator, ',') {
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
	}

	return false
}

// ApplyFilters for filtering or convert value.
func (r *Rule) ApplyFilters(val interface{}, v *Validation) (interface{}, bool) {
	var err error

	// no args filters
	for filter := range r.filters {
		fn := v.FilerFunc(filter)
		if val, err = callFilter(fn, val); err != nil {
			v.WithError(err)
			return nil, false
		}
	}

	// has args filters
	for filter, args := range r.argFilters {
		fn := v.FilerFunc(filter)
		if val, err = callFilter(fn, val, args...); err != nil {
			v.WithError(err)
			return nil, false
		}
	}

	return val, true
}

func callFilter(fn, val interface{}, args ...interface{}) (interface{}, error) {
	var rs []reflect.Value
	if len(args) > 0 {
		rs = Call(fn, buildArgs(val, args)...)
	} else {
		rs = Call(fn, val)
	}

	rl := len(rs)

	// return new val.
	if rl > 0 {
		val = rs[0].Interface()

		if rl == 2 {
			// filter func report error
			if err := rs[1].Interface().(error); err != nil {
				return nil, err
			}
		}
	}

	return val, nil
}

// FilterRule definition
type FilterRule struct {
	// filter list. eg. "int" "str2arr:,"
	filters []string
}
