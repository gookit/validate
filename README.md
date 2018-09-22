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

## Validate Map

```go
package main

import "fmt"
import "time"
import "github.com/gookit/validate"

func main()  {
	m := map[string]interface{}{
		"name":  "inhere",
		"age":   100,
		"oldSt": 1,
		"newSt": 2,
		"email": "some@email.com",
	}

	v := validate.Map(m)
	// v := validate.New(m)
	v.AddRule("name", "required")
	v.AddRule("name", "minLen", 7)
	v.AddRule("age", "max", 99)
	v.AddRule("age", "min", 1)
	v.AddRule("email", "email")

	// v.WithScenes(SValues{
	//	 "create": []string{"name", "email"},
	//	 "update": []string{"name"},
	// })
	
	if v.Validate() { // validate ok
		// do something ...
	} else {
		fmt.Println(v.Errors) // all error messages
		fmt.Println(v.Errors.One()) // returns a random error message text
	}
}
```

## Validate Struct

```go
package main

import "fmt"
import "time"
import "github.com/gookit/validate"

// UserForm struct
type UserForm struct {
	Name     string    `json:"name" validate:"required|minLen:7"`
	Email    string    `json:"email" validate:"email"`
	CreateAt int       `json:"createAt" validate:"min:1"`
	Safe     int       `json:"safe" validate:"-"`
	UpdateAt time.Time `json:"updateAt" validate:"required"`
	Code     string    `json:"code" validate:"customValidator"`
}

// custom validator in the source struct.
func (f *UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}

// Messages you can custom define validator error messages. 
func (f *UserForm) Messages() map[string]string {
	return validate.SMap{
		"required": "oh! the {field} is required",
		"Name.required": "message for special field",
	}
}

// Translates you can custom field translates. 
func (f *UserForm) Translates() map[string]string {
	return validate.SMap{
		"Name": "User Name",
		"Email": "User Email",
	}
}

func main() {
	u := &UserForm{
        Name: "inhere",
    }
	
	v := validate.Struct(u)
	// v := validate.New(u)

	if v.Validate() { // validate ok
		// do something ...
	} else {
		fmt.Println(v.Errors) // all error messages
		fmt.Println(v.Errors.One()) // returns a random error message text
	}
}
```

## Validate Request

```go
package main

import "fmt"
import "time"
import "net/http"
import "github.com/gookit/validate"

func main()  {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := validate.FromRequest(r)
		if err != nil {
			panic(err)
		}
		
		v := data.Create()
		if v.Validate() { // validate ok
			// do something ...
		} else {
			fmt.Println(v.Errors) // all error messages
			fmt.Println(v.Errors.One()) // returns a random error message text
		}
	})
	
	http.ListenAndServe(":8090", handler)
}
```

## Reference

- https://github.com/albrow/forms
- https://github.com/asaskevich/govalidator

## License

**MIT**