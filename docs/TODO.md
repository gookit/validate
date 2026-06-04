# TODO

- [x] cache struct reflect type, tags — done in v1.6.0 (`cache.go` typeMeta + static rule templates)
- [x] field value struct — done in v1.6.0 (unexported `fieldValue`); moved to `internal/fieldval` in v2.0
- [x] use pool for Validation — `Val`/`Var` pooled in v1.6.0; opt-in `Factory`/pool for `Struct`/`Map` landed in v2.0 (`factory.go`: `NewFactory` + `Factory.Struct/Map` + `(*Validation).Release()`)
- [x] inner validators always use `reflect.Value` as param — done in v2.0: `Gt`/`Lt`/`Between` etc. unified through `valueCompare` (`validators_compare.go`)
- [x] can register custom type — done in v2.0 (`register_type.go`: `AddCustomType` / `ResetCustomTypes`)
- field value struct (original sketch, realized as unexported `fieldValue`):

```go
type FieldValue struct {
	name string // field name
	src any // source value
	rv reflect.Value // rv = reflect.ValueOf(src)
	rt reflect.Type // rt = rv.Type()
	real reflect.Value // real = reflect.Indirect(rv)
}

```

## add more validators

- [x] `notIn` — already exists (`validators_compare.go` `NotIn`, registered as `notIn`/`not_in`)
- [ ] `unique/distinct` For arrays & slices, unique will ensure that there are no duplicates. For maps, unique will ensure that there are no duplicate values.

- [x] DateFormat	验证是否是 date, 并且是指定的格式	['publishedAt', 'dateFormat', 'Y-m-d']
- [ ] DateEquals	验证是否是 date, 并且是否是等于给定日期	['publishedAt', 'dateEquals', '2017-05-12'] （`validators_compare.go` 中已留占位，尚未实现/注册）
- [x] BeforeDate	验证字段值必须是给定日期之前的值(ref laravel)	['publishedAt', 'beforeDate', '2017-05-12']
- [x] BeforeOrEqualDate	字段值必须是小于或等于给定日期的值(ref laravel)	['publishedAt', 'beforeOrEqualDate', '2017-05-12']
- [x] AfterOrEqualDate	字段值必须是大于或等于给定日期的值(ref laravel)	['publishedAt', 'afterOrEqualDate', '2017-05-12']
- [x] AfterDate
