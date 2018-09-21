package validate

import (
	"reflect"
	"strconv"
	"strings"
)

// filters.go: filter and convert data

/*************************************************************
 * global filters
 *************************************************************/

var filters map[string]interface{}

// AddFilters
func AddFilters(m GMap) {
	for name, filterFunc := range m {
		AddFilter(name, filterFunc)
	}
}

// AddFilter to the pkg.
func AddFilter(name string, filterFunc interface{}) {
	if filterFunc == nil || reflect.TypeOf(filterFunc).Kind() != reflect.Func {
		panic("validate: invalid validator func, it must be an func")
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

// GetFilter by name
func (v *Validation) GetFilter(name string) interface{} {
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

func ToInt(str string) (int, error) {
	return strconv.Atoi(Trim(str))
}

func MustInt(str string) int {
	val, _ := strconv.Atoi(Trim(str))
	return val
}

func ToUint(str string) (uint64, error) {
	return strconv.ParseUint(Trim(str), 10, 0)
}

func MustUint(str string) uint64 {
	val, _ := strconv.ParseUint(Trim(str), 10, 0)
	return val
}

func ToFloat(str string) (float64, error) {
	return strconv.ParseFloat(Trim(str), 0)
}

func MustFloat(str string) float64 {
	val, _ := strconv.ParseFloat(Trim(str), 0)
	return val
}

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

// StrValue definition
type StrValue string

// Int convert.
func (s StrValue) Int() (val int) {
	if s == "" {
		return 0
	}

	val, _ = strconv.Atoi(s.Trimmed())
	return
}

// Uint convert.
func (s StrValue) Uint() uint {
	if s == "" {
		return 0
	}

	val, _ := strconv.Atoi(s.Trimmed())
	return uint(val)
}

// Int64 convert.
func (s StrValue) Int64() int64 {
	if s == "" {
		return 0
	}

	val, _ := strconv.ParseInt(s.Trimmed(), 10, 64)
	return val
}

// Bool convert.
func (s StrValue) Bool() bool {
	if s == "" {
		return false
	}

	val, _ := strconv.ParseBool(s.Trimmed())
	return val
}

// Float convert. to float 64
func (s StrValue) Float() float64 {
	if s == "" {
		return 0
	}

	val, _ := strconv.ParseFloat(s.Trimmed(), 64)
	return val
}

// Trimmed string
func (s StrValue) Trimmed() string {
	return strings.TrimSpace(string(s))
}

// Split string to slice
func (s StrValue) Split(sep string) (ss []string) {
	if s == "" {
		return
	}

	return stringSplit(s.String(), sep)
}

// String get
func (s StrValue) String() string {
	return string(s)
}
