package validate

import (
	"strings"

	"github.com/gookit/goutil/maputil"
)

// ValidResult carries the outcome of a single validation, fully decoupled from
// the Validation config instance that produced it.
//
// Why it exists: a Validation holds BOTH config (rules/data/validators) and
// result (Errors/safeData/filteredData). Because the result is read AFTER
// Validate() returns, the config instance had to stay alive until the result was
// consumed — which blocked returning it to a pool. ValidResult holds only the
// result, so the config instance can be released immediately (see ValidateR /
// Check) and the caller keeps just the result, with zero lifecycle burden.
//
// Obtain one via Validation.ValidateR() (any configured instance) or the
// top-level Check() (recommended default, internally pooled).
type ValidResult struct {
	// Errors collected during validation. Empty (len 0) when validation passed.
	// A public field to mirror the historical Validation.Errors; ergonomic
	// helpers IsOK()/Fail()/Err() are provided instead of an Errors() method
	// (which would collide with the field name).
	Errors Errors

	// validated safe data. mirrors the old Validation.safeData.
	safeData M
	// filtered clean data. mirrors the old Validation.filteredData.
	filteredData M
}

// IsOK reports whether validation passed (no errors).
func (r *ValidResult) IsOK() bool { return r.Errors.Empty() }

// Fail reports whether validation failed (has at least one error).
func (r *ValidResult) Fail() bool { return !r.Errors.Empty() }

// Err returns a (random) error if validation failed, otherwise nil.
func (r *ValidResult) Err() error { return r.Errors.OneError() }

// SafeData returns all validated safe data.
func (r *ValidResult) SafeData() M { return r.safeData }

// Safe gets a safe value by key.
func (r *ValidResult) Safe(key string) (val any, ok bool) {
	val, ok = r.safeData[key]
	return
}

// SafeVal gets a safe value by key (without the exist flag).
func (r *ValidResult) SafeVal(key string) any { return r.safeData[key] }

// FilteredData returns all filtered data.
func (r *ValidResult) FilteredData() M { return r.filteredData }

// Filtered gets a filtered value by key.
func (r *ValidResult) Filtered(key string) any { return r.filteredData[key] }

// BindStruct binds the safe data onto a struct pointer. alias of BindSafeData.
func (r *ValidResult) BindStruct(ptr any) error { return r.BindSafeData(ptr) }

// BindSafeData binds the safe data onto a struct pointer.
func (r *ValidResult) BindSafeData(ptr any) error {
	if len(r.safeData) == 0 { // no safe data.
		return nil
	}

	// to json bytes
	bts, err := Marshal(expandSafeData(r.safeData))
	if err != nil {
		return err
	}
	return Unmarshal(bts, ptr)
}

// expandSafeData returns safeData ready for binding. When a key carries a dot
// path (eg "address.street", from a nested/bracket form field, #324), it is
// expanded into a nested map so it binds onto nested struct fields. When no key
// contains a dot (the common case) the original flat map is returned unchanged,
// keeping the marshal output byte-identical.
func expandSafeData(safeData M) M {
	hasDot := false
	for k := range safeData {
		if strings.IndexByte(k, '.') >= 0 {
			hasDot = true
			break
		}
	}
	if !hasDot {
		return safeData
	}

	nested := make(map[string]any, len(safeData))
	for k, val := range safeData {
		// on a path/leaf conflict, keep the flat key rather than dropping data.
		if err := maputil.SetByPath(&nested, k, val); err != nil {
			nested[k] = val
		}
	}
	return nested
}
