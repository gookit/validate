package validate

import (
	"github.com/gookit/filter"
	"reflect"
	"strings"
)

var filterAliases = map[string]string{
	"toInt":     "int",
	"str2arr":   "strToArray",
	"str2array": "strToArray",
	"strToArr":  "strToArray",
	"str2time":  "strToTime",
	"trimSpace": "trim",
	"lcFirst":   "lowerFirst",
	"ucFirst":   "upperFirst",
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
	"trim":  reflect.ValueOf(filter.Trim),
	"ltrim": reflect.ValueOf(strings.TrimLeft),
	"rtrim": reflect.ValueOf(strings.TrimRight),
	"email": reflect.ValueOf(filter.Email),
	// change first case.
	"lowerFirst": reflect.ValueOf(filter.LowerFirst),
	"upperFirst": reflect.ValueOf(filter.UpperFirst),
	// change string case.
	"lower": reflect.ValueOf(strings.ToLower),
	"upper": reflect.ValueOf(strings.ToUpper),
	"title": reflect.ValueOf(strings.ToTitle),
	// string clear
	"encodeUrl":  reflect.ValueOf(filter.UrlEncode),
	"escapeJS":   reflect.ValueOf(filter.EscapeJS),
	"escapeHTML": reflect.ValueOf(filter.EscapeHTML),
	// string to array/time
	"strToArray": reflect.ValueOf(filter.StrToArray),
	"strToTime": reflect.ValueOf(filter.StrToTime),
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
