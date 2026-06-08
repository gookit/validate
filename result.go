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

// defaultFactory backs Check() with a package-level pool of reusable
// *Validation instances. Reusing instances across calls amortizes the
// per-instance construction cost; ValidResult decouples the result so the
// instance can be returned to this pool right after Validate.
var defaultFactory = NewFactory()

// Check is the recommended default entry for validating a STRUCT. It returns a
// *ValidResult carrying the outcome (errors + safe/filtered data), decoupled from
// the validation instance.
//
// It is the pooled fast path: structPtr is validated on a *Validation reused from
// a package-level pool (amortizing construct/parse cost), and that instance is
// returned to the pool automatically — no manual lifecycle / Release. This
// mirrors go-playground's validate.Struct(s) usage shape.
//
// Scope / choosing an entry:
//   - struct, need data or binding  -> Check (this) — r.SafeData() / r.BindStruct(&out)
//   - struct, only need pass/fail   -> CheckErr (fewer allocations)
//   - map / programmatic rules / request -> New / Map / FromRequest, then ValidateR()
//
// structPtr MUST be a struct (or pointer to struct); other inputs return a result
// whose Errors carries ErrInvalidData (same as validate.Struct). The pooling and
// the safe-data write-back semantics it relies on are struct-only — hence Check /
// CheckErr do not accept map/form sources.
//
//	r := validate.Check(&user)
//	if r.Fail() { return r.Err() }
//	r.BindSafeData(&out)
func Check(structPtr any, scene ...string) *ValidResult {
	return defaultFactory.Struct(structPtr, scene...).ValidateR()
}

// CheckErr is the opt-in FAST pass/fail entry for a STRUCT: it returns only an
// error (nil = passed; otherwise a random field error via Errors.OneError).
//
// Like Check it is pooled, but it additionally SKIPS collecting safe/filtered
// data and SKIPS building a *ValidResult — so it allocates the least of all
// entries. Use it for hot "accept or reject" paths (e.g. middleware) where the
// cleaned data and BindStruct are not needed.
//
// When you need the cleaned data or struct binding, use Check / ValidateR
// instead. CheckErr is STRUCT-ONLY by design: its skip-collect fast path relies
// on struct source value write-back (UpdateSource) for cross-field correctness,
// which map/form sources do not provide — for those use New / Map + ValidateErr.
//
//	if err := validate.CheckErr(&user); err != nil {
//		return err
//	}
func CheckErr(structPtr any, scene ...string) error {
	v := defaultFactory.Struct(structPtr, scene...)
	v.skipCollect = true // must precede Validate so applyField skips collection
	v.Validate()
	err := v.Errors.OneError()
	v.Release()
	return err
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
