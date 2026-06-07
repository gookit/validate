package validate

import "sync"

// Factory is an opt-in pooled Validation factory. It reuses *Validation
// instances from a sync.Pool across many same-type validations, amortizing the
// per-instance construction cost (newEmpty + translator + rule instantiate).
//
// It is NON-default: validate.New / validate.Struct / validate.Map and their
// lifecycle are unchanged. Only callers who explicitly opt in via a Factory get
// pooling. Most callers should prefer the top-level Check(), which uses a
// package pool automatically; a Factory is for when you want a PRIVATE pool.
//
// Usage:
//
//	f := validate.NewFactory()
//	for _, u := range users {
//		// ValidateR snapshots the result into a ValidResult and returns the
//		// instance to the pool automatically (no manual Release):
//		r := f.Struct(&u).ValidateR()
//		if r.Fail() { /* use r.Errors */ }
//		// r.SafeData() / r.BindStruct(&out) ...
//	}
//
// When you only need the boolean / error face, call Validate() then Release():
//
//	v := f.Struct(&u)
//	if !v.Validate() { /* use v.Errors */ }
//	v.Release() // fully Reset + return to pool
//
// A Validation obtained from a Factory behaves identically to validate.Struct /
// validate.Map (same rules, same result). Do NOT keep using a *Validation after
// Release()/ValidateR(): it may be handed to another caller.
type Factory struct {
	pool *sync.Pool
}

// NewFactory creates a Factory with its own private pool. Each pooled instance
// starts life as a newEmpty()-level blank Validation (same initial state as the
// default path's NewValidation).
func NewFactory() *Factory {
	return &Factory{
		pool: &sync.Pool{
			New: func() any { return newEmpty() },
		},
	}
}

// get fetches a clean instance from the pool and tags it with this factory's
// pool so Release() can return it. resetForReuse() guards against any residual
// state (pooled instances are always reset on Release, but a freshly-New'd
// instance from pool.New is already clean — the reset is cheap and defensive).
func (f *Factory) get() *Validation {
	v := f.pool.Get().(*Validation)
	v.resetForReuse()
	v.pool = f.pool
	return v
}

// Struct gets a pooled Validation assembled from the struct s, equivalent to
// validate.Struct(s, scene...). Call v.Release() when done.
func (f *Factory) Struct(s any, scene ...string) *Validation {
	v := f.get()

	// reuse the pooled StructData carried on v (avoids a new StructData +
	// fieldNames map per call). fromStruct() reset()s it before refilling, and
	// resetForReuse() unbinds it on release — so no source leak / cross-type bleed.
	if v.sd == nil {
		v.sd = &StructData{}
	}
	err := v.sd.fromStruct(s)
	v.data = v.sd
	// assemble rules/config onto the pooled instance (mirrors StructData.Create).
	v.sd.createInto(v, err)
	return v.SetScene(scene...)
}

// Map gets a pooled Validation for the map m, equivalent to validate.Map(m,
// scene...). Rules are added by the caller (e.g. v.StringRules). Call
// v.Release() when done.
func (f *Factory) Map(m map[string]any, scene ...string) *Validation {
	v := f.get()
	v.data = FromMap(m)
	return v.SetScene(scene...)
}

// Release fully resets the Validation and returns it to its owning Factory pool.
// It is a no-op for instances NOT obtained from a Factory (v.pool == nil), so it
// is always safe to call. After Release() the instance must not be used again.
func (v *Validation) Release() {
	if v.pool != nil {
		p := v.pool
		v.resetForReuse()
		v.pool = nil
		p.Put(v)
	}
}
