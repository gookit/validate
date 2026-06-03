# CHANGE LOG

## v1.6.0

Internal performance refactor. **No public API changes** — drop-in upgrade.

### Performance

- Cache per-type reflection metadata (field index, tags, kinds, interface
  checks) in a concurrency-safe type cache; access struct fields via
  `FieldByIndex` instead of `FieldByName`.
- Cache the parsed rule template for "static" struct types (no
  ptr-to-struct / slice-of-struct / map-of-struct) and clone it per
  validation, skipping the per-instance tag walk and rule parsing. Dynamic
  types keep the original per-value path.
- Share built-in error messages instead of copying ~150 entries into every
  `Translator`; resolve with a custom→builtin fallback.
- Build the built-in context validators (`required*`, `*Field`, file checks)
  lazily on first use instead of eagerly per instance.

Measured (Go 1.25, count=6): static struct validation **~-78% time / -74%
allocations** (10403→2295 ns/op, 132→34 allocs/op); map validation -65% /
-42%; slice-of-struct -37% / -20%. Behavior is byte-identical, guarded by a
rule-collection golden test.

### Fixes

- Fix a stack overflow when building metadata for self-referential or
  mutually-recursive struct types.
- Fix a data race in `Val`/`Var`: concurrent calls shared one validation
  instance and raced on its error/validator maps; each call now uses a
  pooled instance.

## V2 - TODO

- [ ] inner validators always use reflect.Value as param. 

**old:**

```go
// Gt check value greater dst value. only check for: int(X), uint(X), float(X)
func Gt(val any, dstVal int64) bool {
```

**v2 new:**

```go
// Gt check value greater dst value. only check for: int(X), uint(X), float(X)
func Gt(val, dstVal any) bool {
	return gt(reflect.ValueOf(val), reflect.ValueOf(dstVal))
}

// internal implements
func gt(val reflect.Value, dstVal reflect.Value) bool
```

- can register custom type
- use sync.Pool for optimize create Validation.

```go
// Validation definition
type Validation struct {
	// for optimize create instance. refer go-playground/validator
	v *Validation
	pool *sync.Pool
    
    // ...
}

	v.pool = &sync.Pool{
		New: func() any {
			return &Validation{
				v: v,
			}
		},
	}
```

- all data operate move to DataSource

```go
type DataFace interface {
	BindStruct() error
	SafeVal(field string) any

...
}

```