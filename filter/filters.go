package filter

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

/*************************************************************
 * global filters
 *************************************************************/

// filters global user filters
var filters map[string]interface{}

// var filterValues map[string]reflect.Value

// AddFilters add global filters
func AddFilters(m map[string]interface{}) {
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

// Filter get filter by name
func Filter(name string) (fn interface{}, ok bool) {
	fn, ok = filters[name]
	return
}

/*************************************************************
 * built in filters
 *************************************************************/

// Trim string
func Trim(str string, cutSet ...string) string {
	if len(cutSet) > 0 {
		return strings.Trim(str, cutSet[0])
	}

	return strings.TrimSpace(str)
}

// Int convert
func Int(str string) (int, error) {
	return ToInt(str)
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

// Uint convert
func Uint(str string) (uint64, error) {
	return ToUint(str)
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

// Str2Array split string to array.
func Str2Array(str string, sep ...string) []string {
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

func panicf(format string, args ...interface{}) {
	panic("filter: " + fmt.Sprintf(format, args...))
}
