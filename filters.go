package validate

import (
	"reflect"
	"strconv"
	"strings"
)

// filters.go: filter and convert data

// Filtration definition. Sanitization Sanitizing Sanitize
type Filtration struct {
	// filtered data
	filtered GMap
}

// Get value by key
func (f *Filtration) Get(key string) (interface{}, bool) {
	return GetByPath(key, f.filtered)
}

// Set value by key
func (f *Filtration) Set(field string, val interface{}) error {
	panic("implement me")
}

/*************************************************************
 * global filters
 *************************************************************/

var filters map[string]interface{}

// var filterValues map[string]reflect.Value

// AddFilters add global filters
func AddFilters(m GMap) {
	for name, filterFunc := range m {
		AddFilter(name, filterFunc)
	}
}

// AddFilter add global filter to the pkg.
func AddFilter(name string, filterFunc interface{}) {
	if filterFunc == nil || reflect.TypeOf(filterFunc).Kind() != reflect.Func {
		panicf("'%s' invalid filter func, it must be an func type", name)
	}

	if filters == nil {
		filters = make(map[string]interface{})
	}

	filters[name] = filterFunc
}

/*************************************************************
 * filters for current validation
 *************************************************************/

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

	if filterFunc == nil || reflect.TypeOf(filterFunc).Kind() != reflect.Func {
		panic("validate: invalid filter func, it must be an func type")
	}

	v.filterFuncs[name] = filterFunc
}

// FilerFunc get filter by name
func (v *Validation) FilerFunc(name string) interface{} {
	if fn, ok := v.filterFuncs[name]; ok {
		return fn
	}

	if fn, ok := filters[name]; ok {
		return fn
	}

	panic("validate: not exists of the filter " + name)
}

// HasFilter check
func (v *Validation) HasFilter(name string) bool {
	if _, ok := v.filterFuncs[name]; ok {
		return true
	}

	_, ok := filters[name]
	return ok
}

/*************************************************************
 * built in filters
 *************************************************************/

// Trim string
func Trim(str string) string {
	return strings.TrimSpace(str)
}

// ToInt convert
func ToInt(str string) (int, error) {
	return strconv.Atoi(Trim(str))
}

// MustInt convert
func MustInt(str string) int {
	val, _ := strconv.Atoi(Trim(str))
	return val
}

// ToUint convert
func ToUint(str string) (uint64, error) {
	return strconv.ParseUint(Trim(str), 10, 0)
}

// MustUint convert
func MustUint(str string) uint64 {
	val, _ := strconv.ParseUint(Trim(str), 10, 0)
	return val
}

// ToInt64 convert
func ToInt64(str string) (int64, error) {
	return strconv.ParseInt(Trim(str), 10, 0)
}

// ToFloat convert
func ToFloat(str string) (float64, error) {
	return strconv.ParseFloat(Trim(str), 0)
}

// MustFloat convert
func MustFloat(str string) float64 {
	val, _ := strconv.ParseFloat(Trim(str), 0)
	return val
}

// ToArray string split to array.
func ToArray(str string, sep ...string) []string {
	if len(sep) > 0 {
		return stringSplit(str, sep[0])
	}

	return stringSplit(str, ",")
}

func stringSplit(str, sep string) (ss []string) {
	str = strings.TrimSpace(str)
	if str == "" {
		return
	}

	for _, val := range strings.Split(str, sep) {
		if val = strings.TrimSpace(val); val != "" {
			ss = append(ss, val)
		}
	}

	return
}

// String definition.
type String string

// CanInt convert.
func (s String) CanInt() bool {
	if s == "" {
		return false
	}

	_, err := strconv.Atoi(s.Trimmed())
	return err == nil
}

// Int convert.
func (s String) Int() (val int) {
	if s == "" {
		return 0
	}

	val, _ = strconv.Atoi(s.Trimmed())
	return
}

// Uint convert.
func (s String) Uint() uint {
	if s == "" {
		return 0
	}

	val, _ := strconv.Atoi(s.Trimmed())
	return uint(val)
}

// Int64 convert.
func (s String) Int64() int64 {
	if s == "" {
		return 0
	}

	val, _ := strconv.ParseInt(s.Trimmed(), 10, 64)
	return val
}

// Bool convert.
func (s String) Bool() bool {
	if s == "" {
		return false
	}

	val, _ := strconv.ParseBool(s.Trimmed())
	return val
}

// Float convert. to float 64
func (s String) Float() float64 {
	if s == "" {
		return 0
	}

	val, _ := strconv.ParseFloat(s.Trimmed(), 0)
	return val
}

// Trimmed string
func (s String) Trimmed() string {
	return strings.TrimSpace(string(s))
}

// Split string to slice
func (s String) Split(sep string) (ss []string) {
	if s == "" {
		return
	}

	return stringSplit(s.String(), sep)
}

// String get
func (s String) String() string {
	return string(s)
}
