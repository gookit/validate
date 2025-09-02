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