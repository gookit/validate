package validate

import (
	"github.com/gookit/filter"
	"reflect"
	"strings"
)

var filterAliases = map[string]string{
	"toInt":   "int",
	"toUint":  "uint",
	"toInt64": "int64",
	"toBool":  "bool",
	"camel":   "camelCase",
	"snake":   "snakeCase",
	//
	"lcFirst":    "lowerFirst",
	"ucFirst":    "upperFirst",
	"ucWord":     "upperWord",
	"trimSpace":  "trim",
	"uppercase":  "upper",
	"lowercase":  "lower",
	"escapeJs":   "escapeJS",
	"escapeHtml": "escapeHTML",
	//
	"str2arr":   "strToArray",
	"str2array": "strToArray",
	"strToArr":  "strToArray",
	"str2time":  "strToTime",
}

// FilterName get real filter name.
func FilterName(name string) string {
	if rName, ok := filterAliases[name]; ok {
		return rName
	}

	return name
}

/*************************************************************
 * Global filters
 *************************************************************/

var filterFuncs map[string]interface{}
var filterValues = map[string]reflect.Value{
	"int":   reflect.ValueOf(filter.Int),
	"uint":  reflect.ValueOf(filter.Uint),
	"int64": reflect.ValueOf(filter.Int64),
	"trim":  reflect.ValueOf(filter.Trim),
	"ltrim": reflect.ValueOf(strings.TrimLeft),
	"rtrim": reflect.ValueOf(strings.TrimRight),
	"email": reflect.ValueOf(filter.Email),
	// change string case.
	"lower":  reflect.ValueOf(strings.ToLower),
	"upper":  reflect.ValueOf(strings.ToUpper),
	"title":  reflect.ValueOf(strings.ToTitle),
	"substr": reflect.ValueOf(filter.Substr),
	// change first case.
	"lowerFirst": reflect.ValueOf(filter.LowerFirst),
	"upperFirst": reflect.ValueOf(filter.UpperFirst),
	// camel <=> snake
	"camelCase": reflect.ValueOf(filter.CamelCase),
	"snakeCase": reflect.ValueOf(filter.SnakeCase),
	"upperWord": reflect.ValueOf(filter.UpperWord),
	// string clear
	"encodeUrl":  reflect.ValueOf(filter.UrlEncode),
	"decodeUrl":  reflect.ValueOf(filter.UrlDecode),
	"escapeJS":   reflect.ValueOf(filter.EscapeJS),
	"escapeHTML": reflect.ValueOf(filter.EscapeHTML),
	// string to array/time
	"strToArray": reflect.ValueOf(filter.StrToArray),
	"strToTime":  reflect.ValueOf(filter.StrToTime),
}

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

	if filterFuncs == nil {
		filterFuncs = make(map[string]interface{})
	}

	filterFuncs[name] = filterFunc
	filterValues[name] = fv
}

// FilterFunc get filter func by name
func FilterFunc(name string) (fn interface{}, ok bool) {
	fn, ok = filterFuncs[name]
	return
}

/*************************************************************
 * filters for current validation
 *************************************************************/

// HasFilter check
func (v *Validation) HasFilter(name string) bool {
	if _, ok := v.filterFuncs[name]; ok {
		return true
	}

	name = FilterName(name)
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
	if v.filterFuncs == nil {
		v.filterFuncs = make(map[string]interface{})
	}

	fv := reflect.ValueOf(filterFunc)

	if filterFunc == nil || fv.Kind() != reflect.Func {
		panicf("invalid filter '%s' func, it must be an func type", name)
	}

	v.filterFuncs[name] = filterFunc
	v.filterValues[name] = fv
}

// FilterFuncValue get filter by name
func (v *Validation) FilterFuncValue(name string) reflect.Value {
	if fv, ok := v.filterValues[name]; ok {
		return fv
	}

	name = FilterName(name)
	if fv, ok := filterValues[name]; ok {
		return fv
	}

	panicf("the filter '%s' is not exists ", name)
	return emptyValue
}

// FilterFunc get filter by name
func (v *Validation) FilterFunc(name string) interface{} {
	if fn, ok := v.filterFuncs[name]; ok {
		return fn
	}

	name = FilterName(name)
	if fn, ok := filterFuncs[name]; ok {
		return fn
	}

	// panicf("the filter '%s' is not exists ", name)
	return nil
}

// FilterRule add filter rule.
// Usage:
// 	v.FilterRule("name", "trim")
// 	v.FilterRule("age", "int")
func (v *Validation) FilterRule(fields string, rule string) {
	rule = strings.TrimSpace(rule)
	rules := stringSplit(strings.Trim(rule, "|:"), "|")

	fieldList := stringSplit(fields, ",")
	if len(fieldList) > 0 {
		r := newFilterRule(fieldList)
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
		val, err = applyFilters(val, r.filters, v)
		if err != nil {
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
