package validate

import (
	"reflect"
	"strings"

	"github.com/gookit/filter"
)

/*************************************************************
 * Global filters
 *************************************************************/

var (
	filterValues map[string]reflect.Value
)

// AddFilters add global filters
func AddFilters(m map[string]any) {
	for name, filterFunc := range m {
		AddFilter(name, filterFunc)
	}
}

// AddFilter add global filter to the pkg.
func AddFilter(name string, filterFunc any) {
	if filterValues == nil {
		filterValues = make(map[string]reflect.Value)
	}

	filterValues[name] = checkFilterFunc(name, filterFunc)
}

/*************************************************************
 * filters for current validation
 *************************************************************/

// AddFilters to the Validation
func (v *Validation) AddFilters(m map[string]any) {
	for name, filterFunc := range m {
		v.AddFilter(name, filterFunc)
	}
}

// AddFilter to the Validation.
func (v *Validation) AddFilter(name string, filterFunc any) {
	if v.filterValues == nil {
		v.filterValues = make(map[string]reflect.Value)
	}

	// v.filterFuncs[name] = filterFunc
	v.filterValues[name] = checkFilterFunc(name, filterFunc)
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
//
// Usage:
//
//	v.FilterRule("name", "trim|lower")
//	v.FilterRule("age", "int")
func (v *Validation) FilterRule(field string, rule string) *FilterRule {
	rule = strings.TrimSpace(rule)
	rules := stringSplit(strings.Trim(rule, "|:"), "|")
	fields := stringSplit(field, ",")

	if len(fields) == 0 || len(rules) == 0 {
		panicf("no enough arguments or contains invalid argument for add filter rule")
	}

	r := newFilterRule(fields)
	r.AddFilters(rules...)
	v.filterRules = append(v.filterRules, r)

	return r
}

// FilterRules add multi filter rules.
func (v *Validation) FilterRules(rules map[string]string) *Validation {
	for field, rule := range rules {
		v.FilterRule(field, rule)
	}
	return v
}

/*************************************************************
 * filtering rule
 *************************************************************/

// FilterRule definition
type FilterRule struct {
	// fields to filter
	fields []string
	// filter name list
	filters []string
	// filter args. { index: "args" }
	filterArgs map[int]string
}

func newFilterRule(fields []string) *FilterRule {
	return &FilterRule{
		fields: fields,
		// init map
		filterArgs: make(map[int]string),
	}
}

// AddFilters add filter(s).
//
// Usage:
//
//	r.AddFilters("int", "str2arr:,")
func (r *FilterRule) AddFilters(filters ...string) *FilterRule {
	for _, filterName := range filters {
		pos := strings.IndexRune(filterName, ':')
		if pos > 0 { // has filter args
			name := filterName[:pos]
			index := len(r.filters)
			r.filters = append(r.filters, name)
			r.filterArgs[index] = filterName[pos+1:]
		} else {
			r.filters = append(r.filters, filterName)
		}
	}

	return r
}

// Apply rule for the rule fields
func (r *FilterRule) Apply(v *Validation) (err error) {
	// filter field value
	for _, field := range r.Fields() {
		val, exist, zero := v.tryGet(field)
		if !exist || zero {
			defVal, ok := v.GetDefValue(field)
			// there is also no custom default value
			if !ok {
				continue
			}

			// update source data field value
			newVal, err := v.updateValue(field, defVal)
			if err != nil {
				return err
			}

			// re-set value
			val = newVal

			// dont need check default value
			if !v.CheckDefault {
				v.safeData[field] = newVal // save validated value.
				continue
			}
		}

		// interior ".*" wildcard path (eg "ports.*.container_start"): the value
		// fetched via GetByPath is a slice []any{...} of the per-element scalars
		// collected from after the wildcard. apply the filters to EACH element
		// (mirrors validateWildcardSlice on the validate side); otherwise a scalar
		// filter like "int" would try to convert the whole slice as a single value
		// and fail with "convert value type error" (#262). the per-element result
		// is stored back under the wildcard key, which the validate path reads via
		// tryGet -> filteredData, so element-wise validation just works.
		//
		// NOTE: only INTERIOR wildcards expand. a TERMINAL ".*" (eg "Domains.*" on
		// a []string field) fetches the field's own slice, where slice-aware
		// filters like "trimStrings"/"unique" operate on the whole slice (#172) —
		// those keep the scalar/whole-value path below.
		if strings.Contains(field, ".*") && !strings.HasSuffix(field, ".*") {
			if rv := reflect.ValueOf(val); rv.Kind() == reflect.Slice {
				out := make([]any, rv.Len())
				for i := 0; i < rv.Len(); i++ {
					if out[i], err = r.applyFilters(v, rv.Index(i).Interface()); err != nil {
						return err
					}
				}
				// note: source write-back (updateValue) is skipped for the wildcard
				// path — it only supports whole-field Set; the collected slice in
				// filteredData carries the result (covers Map/JSON sources).
				v.filteredData[field] = out
				continue
			}
			// not a slice (single match) -> fall through to the scalar path below.
		}

		// call filters
		if val, err = r.applyFilters(v, val); err != nil {
			return err
		}

		// update source data field value
		newVal, err := v.updateValue(field, val)
		if err != nil {
			return err
		}

		// save filtered value.
		v.filteredData[field] = newVal
	}
	return
}

// applyFilters runs the rule's full filter chain on a single value and returns
// the filtered result. Used for both the scalar path and per-element on the
// ".*" wildcard path.
func (r *FilterRule) applyFilters(v *Validation, val any) (any, error) {
	var err error
	for i, name := range r.filters {
		fv := v.FilterFuncValue(name)
		args := parseArgString(r.filterArgs[i])
		if !fv.IsValid() { // is built in filters
			val, err = filter.Apply(name, val, args)
		} else {
			val, err = callCustomFilter(fv, val, args)
		}
		if err != nil {
			return nil, err
		}
	}
	return val, nil
}

// Fields name get
func (r *FilterRule) Fields() []string {
	return r.fields
}

func callCustomFilter(fv reflect.Value, val any, args []string) (any, error) {
	var rs []reflect.Value
	if len(args) > 0 {
		rs = CallByValue(fv, buildArgs(val, strings2Args(args))...)
	} else {
		rs = CallByValue(fv, val)
	}

	// return new val.
	if rl := len(rs); rl > 0 {
		val = rs[0].Interface() // `func(val) (newVal)`

		// filter func report error. `func(val) (newVal, error)`
		if rl == 2 {
			err := rs[1].Interface()
			if err != nil {
				return nil, err.(error)
			}
		}
	}

	return val, nil
}
