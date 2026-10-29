# CHANGE LOG

## Unreleased

- Resolve string and integer map keys using their declared type during nested
  struct validation and filter/default write-back, including defined key types.
  Invalid or overflowing integer path segments are treated as missing entries.
- Treat out-of-range slice and array indexes in struct field paths as missing instead of panicking.
- `in`/`not_in` now support a slice or array value: `in:go,php` passes when every element is in the
  enumeration, `not_in:go,php` passes when no element is (a non-empty slice used to fail `in` and pass
  `not_in` regardless of its elements, see #266). A nil element is ignored.
- The `min`/`max` (and `gt`/`gte`/`lt`/`lte`/`between`) comments and the README now say that a string
  value is compared lexicographically, not by its length — `min:3` passes for "ab"; use
  `min_len`/`max_len` for a length check. A test pins that behaviour.
- A struct field can be referenced in lower case now, including an acronym-style name: with the
  field `MD5`, `required_without_all:md5,url` and `Get("md5")` resolve it (before, only the exact
  `MD5` worked, and a cross-field rule treated the name as an empty value, see #347).
- The `in`/`not_in` rules report an unsupported map value with a clear message instead of
  "value must be in the enum [...]". A map element of a slice value is reported the same way.
- `in`/`not_in` now check a JSON array: JSON decodes an array into `[]any`, whose elements
  used to stay wrapped in their interface and failed the rule (`{"tags":["go","php"]}` did
  not pass `in:go,php`). Elements are unwrapped and a `nil` element is ignored (#353).
- `in`/`not_in` now compare a JSON number: it decodes into a `float64`, so `1` matches
  `in:1,2` (it used to fail); a fractional number such as `1.5` still fails.
- The `notIn` rule no longer formats its enum list with `%d`, which rendered as
  "must not be in the given enum list [%!d(string=go) %!d(string=php)]"; it now prints
  "[go php]" (also fixed in the zh-CN, zh-TW and ru-RU messages).
- `required_unless` now requires the value when the other field is missing: an absent field
  cannot equal any of the listed values, so the exemption no longer applies (#354). Before,
  a missing (or unset) dependency field silently exempted the field under validation. A
  non-empty value and the existent numeric-zero behaviour are unchanged.
- `required_if` with a single literal now compares a defined scalar type: with
  `type Status string`, `required_if:Mode,member` requires the field for `Status("member")`
  too, which used to be a mismatch and left an empty field valid (#355).

## v2.0.0

v2.0 keeps breaking changes **intentionally minimal** (5 items). Core API,
validator names, tag semantics and the `DataFace` interface are unchanged — most
projects upgrade by bumping the import path to `/v2`. See
[docs/UPGRADE-v2.md](docs/UPGRADE-v2.md) for the full migration guide.

### Breaking Changes

- **Module path** `github.com/gookit/validate` → `github.com/gookit/validate/v2`
  (per Go Modules semantic import versioning). Update both `go get` and all
  `import` lines.
- **Minimum Go version** 1.19 → **1.21** (`go.mod`).
- **`Between` signature** `Between(val any, min, max int64)` →
  `Between(val, min, max any)`, now consistent with `Gt`/`Lt` via `valueCompare`
  (supports int / uint / float / string). Semantic change for fractional bounds:
  `Between(2.9, 1, 2)` was `true` (int64 truncation) and is now `false` (no more
  truncation). Tag/`StringRule` usage like `between:1,2` is unaffected.
- **Removed deprecated `ValueLen(v)`** — use goutil's `reflects.Len(v)` instead.
- **Sub-struct cascade now requires the parent field to carry a `validate` tag.**
  Previously a sub-struct field (struct / `*struct` / slice-of-struct /
  map-of-struct) was **always** descended into to collect its inner rules. Now
  (`CheckSubOnParentMarked` defaults to **true**) cascade only happens when the
  parent field has a `validate` tag — the value may be **empty** (`validate:""`
  is enough to mark it); a **named** field with **no** `validate` tag is no
  longer descended into (Java `@Valid`-style opt-in). **Anonymous embedded
  structs** (e.g. `type Bar struct { Foo }`) are **exempt** — they are part of
  the parent and always cascade regardless of tag; only **named** sub-struct
  fields require the tag. To restore the v1 "always cascade" behavior globally:
  `validate.Config(func(o *validate.GlobalOption){ o.CheckSubOnParentMarked = false })`.
  See [docs/UPGRADE-v2.md](docs/UPGRADE-v2.md) for migration details.

> Note: the `DataFace` interface is **unchanged** (a redesign was planned during
> v2.0 but rejected after profiling); custom `DataFace` implementers are unaffected.

### New Features

- **`AddCustomType(fn CustomTypeFunc, types ...any)`** — register an underlying
  value extractor for custom/wrapped types (e.g. `sql.NullString`, money types).
  The extracted value flows through the existing validation paths (`required` /
  numeric compare / length / string rules). `CustomTypeFunc func(field
  reflect.Value) any`; returning `nil` is treated as empty. Mirrors
  go-playground/validator's `RegisterCustomTypeFunc`. `ResetCustomTypes()` clears
  the registry. Zero overhead on the hot path when nothing is registered (atomic
  gate short-circuit).
- **`NewFactory()` + `Factory.Struct` / `Factory.Map` + `(*Validation).Release()`**
  — opt-in pooled factory that reuses `*Validation` instances across many
  same-type validations, amortizing construction cost (allocs roughly halved in
  reuse scenarios). Non-default: `Struct` / `Map` / `New` behavior and lifecycle
  are unchanged.

### Performance

Measured on Go 1.25 (count=6) against the v1.6.0 final:

| Benchmark | v1.6.0 | v2.0 default |
|---|---|---|
| StructFlat | 2295 ns / 34 allocs | 1870 ns / 23 allocs (-18% / -32%) |
| StructNested | 2137 ns / 34 allocs | 1722 ns / 24 allocs (-19% / -29%) |
| MapValidate | 3314 ns / 72 allocs | 3350 ns / 72 allocs (flat) |
| ValValue | 725 ns / 11 allocs | 737 ns / 11 allocs (flat) |
| SliceOfStruct | — | 10900 ns / 208 allocs (dynamic, flat) |
| **FactoryStructReuse** (new) | — | 1633 ns / **11 allocs** (-52% allocs vs non-factory) |

Coverage 95.7% (v1.6.0 was 95.6%).

### Internal

- Pre-convert rule args at build time (drop the runtime `convertArgsType`).
- Share immutable static template rules instead of cloning per instance.
- Fix string validators panicking on non-string fields.
- Split `validators.go` by category (compare / string / type files).
- Sink pure-reflect helpers and `fieldValue` into `internal/` (`reflectx`,
  `fieldval`).

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