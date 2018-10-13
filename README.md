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
- Many commonly used validators have been built in(> 50), see [Built In Validators](#built-in-validators)

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

<a id="built-in-filters"></a>
## Built In Filters

filter/aliases | description 
-------------------|-------------------------------------------
`int/integer`  | convert value(string) to int type `v.FilterRule("id", "int")`
`trim`  | clear string.

<a id="built-in-validators"></a>
## Built In Validators

**Notice:**

- `intX` is contains: int, int8, int16, int32, int64
- `uintX` is contains: uint, uint8, uint16, uint32, uint64
- `floatX` is contains: float32, float64

validator/aliases | description
-------------------|-------------------------------------------
`required`  | check value is not empty. 
`int/integer/isInt`  | check value is `intX` `uintX` type
`in/enum`  |  description
`range/between`  |  description
`eq/equal/isEqual`  |  description
`uint/isUint`  |  check value is uint(`uintX`) type, `value >= 0`
`map/isMap`  |  description
`max/lte`  |  description
`intStr/intString/isIntString`  |  description
`ne/notEq/notEqual`  |  description
`gtField`  |  description
`lteField`  |  description
`arr/array/isArray`  |   check value is array type
`lt/lessThan`  |  check value is less than the given size(use for `intX` `uintX` `floatX`)
`gt/greaterThan`  |  check value is greater than the given size(use for `intX` `uintX` `floatX`)
`email/isEmail`  |   check value is email address string.
`eqField`  |  description
`intEq/intEqual`  |  description
`len/length`  |  description
`float/isFloat`  |  check value is float(`floatX`) type
`min/gte`  |  Check value is less than or equal to the given size(for `intX` `uintX` `floatX`)
`neField`  |  description
`slice/isSlice`  |  check value is slice type(`[]intX` `[]uintX` `[]byte` `[]string` ...).
`regex/regexp`  |  description
`ints/isInts`  |  check value is int slice type(only allow `[]int`).
`strings/isStrings`  |  check value is string slice type(only allow `[]string`).
`minLen/minLength`  |  check the minimum length of the value is the given size
`maxLen/maxLength`  |  check the maximum length of the value is the given size
`notIn`  |  description
`gteField`  |  description
`ltField`  |  description
`ip/isIP`  |  check value is IP(v4 or v6) string.
`ipv4/isIPv4`  |  check value is IPv4 string.
`ipv6/isIPv6`  |  check value is IPv6 string.
`bool/isBool`  |  check value is bool string.
`string/isString`  |  description
`hasWhitespace` | check value string has Whitespace.
`ascii/ASCII/isASCII` | check value is ASCII string.
`alpha/isAlpha` | check value is Alpha string.
`alphaNum/isAlphaNum` | check value is AlphaNum string.
`base64/isBase64` | check value is Base64 string.
`CIDR/isCIDR` | check value is CIDR string.
`CIDRv4/isCIDRv4` | check value is CIDRv4 string.
`CIDRv6/isCIDRv6` | check value is CIDRv6 string.
`dnsName/DNSName/isDNSName` | check value is DNSName string.
`dataURI/isDataURI` | check value is DataURI string.
`empty/isEmpty` | check value is Empty string.
`filePath/isFilePath` | check value is FilePath string.
`hexColor/isHexColor` | check value is HexColor string.
`hexadecimal/isHexadecimal` | check value is Hexadecimal string.
`isbn10/ISBN10/isISBN10` | check value is ISBN10 string.
`isbn13/ISBN13/isISBN13` | check value is ISBN13 string.
`json/JSON/isJSON` | check value is JSON string.
`lat/latitude/isLatitude` | check value is Latitude string.
`lon/longitude/isLongitude` | check value is Longitude string.
`mac/isMAC` | check value is MAC string.
`multiByte/isMultiByte` | check value is MultiByte string.
`num/number/isNumber` | check value is number string. `>= 0`
`printableASCII/isPrintableASCII` | check value is PrintableASCII string.
`rgbColor/RGBColor/isRGBColor` | check value is RGBColor string.
`url/isURL` | check value is URL string.
`uuid/isUUID` | check value is UUID string.
`uuid3/isUUID3` | check value is UUID3 string.
`uuid4/isUUID4` | check value is UUID4 string.
`uuid5/isUUID5` | check value is UUID5 string.
`unixPath/isUnixPath` | check value is UnixPath string.
`winPath/isWinPath` | check value is WinPath string.

## Reference

- https://github.com/albrow/forms
- https://github.com/asaskevich/govalidator
- https://github.com/go-playground/validator
- https://github.com/inhere/php-validate

## License

**MIT**