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
- Built-in common data type filter/converter. see [Built In Filters](#built-in-filters)
- Many commonly used validators have been built in(> 60), see [Built In Validators](#built-in-validators)

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

**Notice:**

- `intX` is contains: int, int8, int16, int32, int64
- `uintX` is contains: uint, uint8, uint16, uint32, uint64
- `floatX` is contains: float32, float64

<a id="built-in-filters"></a>
## Built In Filters

> Filters provide by: [gookit/filter](https://github.com/gookit/filter)

filter/aliases | description 
-------------------|-------------------------------------------
`trim/trimSpace`  | Clean up whitespace characters on both sides of the string
`ltrim`  | Clean up whitespace characters on left sides of the string
`rtrim`  | Clean up whitespace characters on right sides of the string
`int/integer`  | convert value(string/intX/floatX) to int type `v.FilterRule("id", "int")`
`uint`  | convert value(string/intX/floatX) to `uint` type `v.FilterRule("id", "uint")`
`int64`  | convert value(string/intX/floatX) to `int64` type `v.FilterRule("id", "int64")`
`bool`  | convert string value to bool. (`true`: "1", "on", "yes", "true", `false`: "0", "off", "no", "false")
`lower/lowercase` | Convert string to lowercase
`upper/uppercase` | Convert string to uppercase
`lcFirst/lowerFirst` | Convert the first character of a string to lowercase
`ucFirst/upperFirst` | Convert the first character of a string to uppercase
`ucWord/upperWord` | Convert the first character of each word to uppercase
`camel/camelCase` | Convert string to camel naming style
`snake/snakeCase` | Convert string to snake naming style
`escapeJs/escapeJS` | escape JS string.
`escapeHtml/escapeHTML` | escape HTML string.
`str2ints/strToInts` | Convert string to int slice `[]int` 
`str2time/strToTime` | Convert date string to `time.Time`.
`str2arr/str2array/strToArray` | Convert string to string slice `[]string`

<a id="built-in-validators"></a>
## Built In Validators

validator/aliases | description
-------------------|-------------------------------------------
`required`  | check value is not empty. 
`-/safe`  | Tag field values ​​are safe and do not require validation
`int/integer/isInt`  | check value is `intX` `uintX` type
`uint/isUint`  |  check value is uint(`uintX`) type, `value >= 0`
`bool/isBool`  |  check value is bool string(`true`: "1", "on", "yes", "true", `false`: "0", "off", "no", "false").
`string/isString`  |  check value is string type.
`float/isFloat`  |  check value is float(`floatX`) type
`slice/isSlice`  |  check value is slice type(`[]intX` `[]uintX` `[]byte` `[]string` ...).
`in/enum`  |  Check if the value is in the given enumeration
`notIn`  |  Check if the value is not in the given enumeration
`range/between`  |  Check that the value is a number and is within the given range
`max/lte`  |  Check value is less than or equal to the given value
`min/gte`  |  Check value is less than or equal to the given size(for `intX` `uintX` `floatX`)
`intStr/intString/isIntString`  |  check value is an int string.
`eq/equal/isEqual`  |  Check that the input value is equal to the given value
`ne/notEq/notEqual`  |  Check that the input value is not equal to the given value
`lt/lessThan`  |  check value is less than the given size(use for `intX` `uintX` `floatX`)
`gt/greaterThan`  |  check value is greater than the given size(use for `intX` `uintX` `floatX`)
`email/isEmail`  |   check value is email address string.
`intEq/intEqual`  |  check value is int and equals to the given value.
`len/length`  |  check value length is equals to the given size(use for `string` `array` `slice` `map`).
`regex/regexp`  |  Check if the value can pass the regular verification
`arr/array/isArray`  |   check value is array type
`map/isMap`  |  Check value is a MAP type
`strings/isStrings`  |  check value is string slice type(only allow `[]string`).
`ints/isInts`  |  check value is int slice type(only allow `[]int`).
`minLen/minLength`  |  check the minimum length of the value is the given size
`maxLen/maxLength`  |  check the maximum length of the value is the given size
`eqField`  |  Check that the field value is equals to the value of another field
`neField`  |  Check that the field value is not equals to the value of another field
`gteField`  |  Check that the field value is greater than or equal to the value of another field
`gtField`  |  Check that the field value is greater than the value of another field
`lteField`  |  Check if the field value is less than or equal to the value of another field
`ltField`  |  Check that the field value is less than the value of another field
`hasWhitespace` | check value string has Whitespace.
`ascii/ASCII/isASCII` | check value is ASCII string.
`alpha/isAlpha` | check value is Alpha string.
`alphaNum/isAlphaNum` | check value is AlphaNum string.
`multiByte/isMultiByte` | check value is MultiByte string.
`base64/isBase64` | check value is Base64 string.
`dnsName/DNSName/isDNSName` | check value is DNSName string.
`dataURI/isDataURI` | check value is DataURI string.
`empty/isEmpty` | check value is Empty string.
`hexColor/isHexColor` | check value is HexColor string.
`hexadecimal/isHexadecimal` | check value is Hexadecimal string.
`json/JSON/isJSON` | check value is JSON string.
`lat/latitude/isLatitude` | check value is Latitude string.
`lon/longitude/isLongitude` | check value is Longitude string.
`mac/isMAC` | check value is MAC string.
`num/number/isNumber` | check value is number string. `>= 0`
`printableASCII/isPrintableASCII` | check value is PrintableASCII string.
`rgbColor/RGBColor/isRGBColor` | check value is RGBColor string.
`url/isURL` | check value is URL string.
`ip/isIP`  |  check value is IP(v4 or v6) string.
`ipv4/isIPv4`  |  check value is IPv4 string.
`ipv6/isIPv6`  |  check value is IPv6 string.
`CIDR/isCIDR` | check value is CIDR string.
`CIDRv4/isCIDRv4` | check value is CIDRv4 string.
`CIDRv6/isCIDRv6` | check value is CIDRv6 string.
`uuid/isUUID` | check value is UUID string.
`uuid3/isUUID3` | check value is UUID3 string.
`uuid4/isUUID4` | check value is UUID4 string.
`uuid5/isUUID5` | check value is UUID5 string.
`filePath/isFilePath` | check value is FilePath string.
`unixPath/isUnixPath` | check value is UnixPath string.
`winPath/isWinPath` | check value is WinPath string.
`isbn10/ISBN10/isISBN10` | check value is ISBN10 string.
`isbn13/ISBN13/isISBN13` | check value is ISBN13 string.

## Reference

- https://github.com/albrow/forms
- https://github.com/asaskevich/govalidator
- https://github.com/go-playground/validator
- https://github.com/inhere/php-validate

## License

**[MIT](LICENSE)**