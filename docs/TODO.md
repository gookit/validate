# TODO

- use pool for Validation
- cache struct reflect type, tags
- field value struct:

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

- `notIn`
- `unique/distinct` For arrays & slices, unique will ensure that there are no duplicates. For maps, unique will ensure that there are no duplicate values.

- DateFormat	验证是否是 date, 并且是指定的格式	['publishedAt', 'dateFormat', 'Y-m-d']
- DateEquals	验证是否是 date, 并且是否是等于给定日期	['publishedAt', 'dateEquals', '2017-05-12']
- BeforeDate	验证字段值必须是给定日期之前的值(ref laravel)	['publishedAt', 'beforeDate', '2017-05-12']
- BeforeOrEqualDate	字段值必须是小于或等于给定日期的值(ref laravel)	['publishedAt', 'beforeOrEqualDate', '2017-05-12']
- AfterOrEqualDate	字段值必须是大于或等于给定日期的值(ref laravel)	['publishedAt', 'afterOrEqualDate', '2017-05-12']
- AfterDate
