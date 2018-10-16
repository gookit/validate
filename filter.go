package validate

import (
	"github.com/gookit/filter"
	"reflect"
	"strings"
)

/*************************************************************
 * Global filters
 *************************************************************/

var filterValues map[string]reflect.Value

// AddFilters add global filters
func AddFilters(m map[string]interface{}) {
	for name, filterFunc := range m {
		AddFilter(name, filterFunc)
	}
}

// AddFilter add global filter to the pkg.
func AddFilter(name string, filterFunc interface{}) {
	fv := reflect.ValueOf(filterFunc)
	if filterFunc == nil || fv.Kind() != reflect.Func {
		panicf("'%s' invalid filter func, it must be an func type", name)
	}

	if filterValues == nil {
		filterValues = make(map[string]reflect.Value)
	}

	filterValues[name] = fv
}

/*************************************************************
 * filters for current validation
 *************************************************************/

// HasFilter check
func (v *Validation) HasFilter(name string) bool {
	if _, ok := v.filterValues[name]; ok {
		return true
	}

	name = filter.Name(name)
	_, ok := filterValues[name]
	return ok
}

// AddFilters to the Validation
func (v *Validation) AddFilters(m map[string]interface{}) {
	for name, filterFunc := range m {
		v.AddFilter(name, filterFunc)
	}
}

// AddFilter to the Validation.
func (v *Validation) AddFilter(name string, filterFunc interface{}) {
	fv := reflect.ValueOf(filterFunc)
	if filterFunc == nil || fv.Kind() != reflect.Func {
		panicf("invalid filter '%s' func, it must be an func type", name)
	}

	if v.filterValues == nil {
		v.filterValues = make(map[string]reflect.Value)
	}

	// v.filterFuncs[name] = filterFunc
	v.filterValues[name] = fv
}

// FilterFuncValue get filter by name
func (v *Validation) FilterFuncValue(name string) reflect.Value {
	if fv, ok := v.filterValues[name]; ok {
		return fv
	}

	if fv, ok := filterValues[name]; ok {
		return fv
	}

	return emptyValue
}

// FilterRule add filter rule.
// Usage:
// 	v.FilterRule("name", "trim")
// 	v.FilterRule("age", "int")
func (v *Validation) FilterRule(field string, rule string) {
	rule = strings.TrimSpace(rule)
	rules := stringSplit(strings.Trim(rule, "|:"), "|")
	fields := stringSplit(field, ",")

	if len(fields) > 0 {
		r := newFilterRule(fields)
		r.AddFilters(rules...)
		v.filterRules = append(v.filterRules, r)
	}
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
	for _, filterName := range filters {
		pos := strings.IndexRune(filterName, ':')

		// has filter args
		if pos > 0 {
			name := filterName[:pos]
			r.filters[name] = filterName[pos+1:]
		} else {
			r.filters[filterName] = ""
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
		for name, argStr := range r.filters {
			fv := v.FilterFuncValue(name)
			args := parseArgString(argStr)

			if !fv.IsValid() {
				val, err = filter.Apply(name, val, args)
			} else {
				val, err = callFilter(fv, val, strings2Args(args))
			}

			if err != nil {
				return  err
			}
		}

		// save filtered value.
		v.filteredData[field] = val
	}

	return
}

// Fields name get
func (r *FilterRule) Fields() []string {
	return r.fields
}

func applyFilters(val interface{}, filters map[string]string, v *Validation) (interface{}, error) {
	var err error
	for name, argStr := range filters {
		fv := v.FilterFuncValue(name)
		args := parseArgString(argStr)

		if !fv.IsValid() {
			val, err = filter.Apply(name, val, args)
		} else {
			val, err = callFilter(fv, val, strings2Args(args))
		}

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
