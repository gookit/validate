# Validate

[![GoDoc](https://godoc.org/github.com/gookit/validate?status.svg)](https://godoc.org/github.com/gookit/validate)
[![Build Status](https://travis-ci.org/gookit/validate.svg?branch=master)](https://travis-ci.org/gookit/validate)
[![Coverage Status](https://coveralls.io/repos/github/gookit/validate/badge.svg?branch=master)](https://coveralls.io/github/gookit/validate?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/validate)](https://goreportcard.com/report/github.com/gookit/validate)

The package is a generic go data validate library.

- Support validate Map, Struct, Request(Form, JSON, url.Values) data
- Support filter/sanitize data before validate
- Support add custom filter/validator func
- Support scene settings, verify different fields in different scenes
- Support custom error messages, field translates.
- Support language messages, built in `en`, `zh-CN`

> Inspired the projects [albrow/forms](https://github.com/albrow/forms) and [asaskevich/govalidator](https://github.com/asaskevich/govalidator). Thank you very much

## Go Doc

- [godoc for gopkg](https://godoc.org/gopkg.in/gookit/validate.v1)
- [godoc for github](https://godoc.org/github.com/gookit/validate)

## Validate Struct

```go
package main

import "fmt"
import "time"
import "github.com/gookit/validate"

// UserForm struct
type UserForm struct {
	Name     string    `validate:"required|minLen:7"`
	Email    string    `validate:"email"`
	Age      int       `validate:"required|int|min:1|max:99"`
	CreateAt int       `validate:"min:1"`
	Safe     int       `validate:"-"`
	UpdateAt time.Time `validate:"required"`
	Code     string    `validate:"customValidator"`
}

// custom validator in the source struct.
func (f UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}

// Messages you can custom validator error messages. 
func (f UserForm) Messages() map[string]string {
	return validate.MS{
		"required": "oh! the {field} is required",
		"Name.required": "message for special field",
	}
}

// Translates you can custom field translates. 
func (f UserForm) Translates() map[string]string {
	return validate.MS{
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
	
	// can also
	v.StringRule("age", "required|int|min:1|max:99")
	v.StringRule("name", "required|minLen:7")

	// v.WithScenes(map[string]string{
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
		// setting rules
		v.FilterRule("age", "int") // convert value to int
		
		v.AddRule("name", "required")
		v.AddRule("name", "minLen", 7)
		v.AddRule("age", "max", 99)
		
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

## Built In Filters

filter/aliases | description 
-------------------|-------------------------------------------
`int/integer`  | convert value(string) to int type `v.FilterRule("id", "int")`

## Built In Validators

validator/aliases | description
-------------------|-------------------------------------------
`required`  | check value is not empty. 
`int/integer/isInt`  | check value is int type
`in/enum`  |  description
`range/between`  |  description
`eq/equal/isEqual`  |  description
`minLen/minLength`  |  description
`uint/isUint`  |  description
`map/isMap`  |  description
`max/lte`  |  description
`ipv4/isIPv4`  |  description
`intStr/intString/isIntString`  |  description
`ne/notEq/notEqual`  |  description
`gtField`  |  description
`lteField`  |  description
`arr/array/isArray`  |  description
`lt`  |  description
`email/isEmail`  |  description
`eqField`  |  description
`intEq/intEqual`  |  description
`len/length`  |  description
`float/isFloat`  |  description
`min/gte`  |  description
`neField`  |  description
`slice/isSlice`  |  description
`regex/regexp`  |  description
`strings/isStrings`  |  description
`ints/isInts`  |  description
`ip/isIP`  |  description
`gt`  |  description
`maxLen/maxLength`  |  description
`notIn`  |  description
`gteField`  |  description
`ltField`  |  description
`ipv6/isIPv6`  |  description
`bool/isBool`  |  description
`string/isString`  |  description

## Reference

- https://github.com/albrow/forms
- https://github.com/asaskevich/govalidator
- https://github.com/go-playground/validator
- https://github.com/inhere/php-validate

## License

**MIT**