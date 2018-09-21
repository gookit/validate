# validate

the package is a generic go data validate library.

Inspired the projects [albrow/forms](https://github.com/albrow/forms) and [asaskevich/govalidator](https://github.com/asaskevich/govalidator). Thank you very much

## validators

```text
v.SetRules()
// 
v.CacheRules("id key")
```

```text
v := Validation.FromRequest(req)

v := Validation.FromMap(map).New()

v := Validation.FromStruct(struct)

v.SetRules(
    v.Required("field0, field1", "%s is required"),
    v.Min("field0, field1", 2),
    v.Max("field1", 5),
    v.IntEnum("field2", []int{1,2}),
    v.StrEnum("field3", []string{"tom", "john"}),
    v.Range("field4", 0, 5),
    // add rule
    v.AddRule("field5", "required;min(1);max(20);range(1,23);gtField(field3)"),
)

if !v.Validate() {
    fmt.Println(v.Errors)
}

// do something ...
```

validate map:

```text
v := validate.Map(map)

v.SetRules(
    v.Required("field0, field1", "%s is required"),
    v.Min("field0, field1", 2),
    v.Max("field1", 5),
)

if !v.Validate() {
    fmt.Println(v.Errors)
}

// do something ...
```

validate struct:

```text
v := validate.Struct(struct)

v.SetRules(
    v.Required("field0, field1", "%s is required"),
    v.Min("field0, field1", 2),
    v.Max("field1", 5),
)

if !v.Validate() {
    fmt.Println(v.Errors)
}

// do something ...
```

design 1:

```text
type StructValidation struct {
    Validation
    Value reflect.Value
}
```