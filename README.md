# Validate

[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/gookit/validate)](https://github.com/gookit/validate)
[![GoDoc](https://godoc.org/github.com/gookit/validate?status.svg)](https://pkg.go.dev/github.com/gookit/validate)
[![Build Status](https://travis-ci.org/gookit/validate.svg?branch=master)](https://travis-ci.org/gookit/validate)
[![Coverage Status](https://coveralls.io/repos/github/gookit/validate/badge.svg?branch=master)](https://coveralls.io/github/gookit/validate?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/validate)](https://goreportcard.com/report/github.com/gookit/validate)
[![Actions Status](https://github.com/gookit/validate/workflows/Unit-Tests/badge.svg)](https://github.com/gookit/validate/actions)

The package is a generic Go data validate and filter tool library.

- Support quick validate `Map`, `Struct`, `Request`(`Form`, `JSON`, `url.Values`, `UploadedFile`) data
- Support filter/sanitize data before validate
- Support add custom filter/validator func
- Support scene settings, verify different fields in different scenes
- Support custom error messages, field translates.
- Customizable i18n aware error messages, built in `en`, `zh-CN`, `zh-TW`
- Built-in common data type filter/converter. see [Built In Filters](#built-in-filters)
- Many commonly used validators have been built in(**> 70**), see [Built In Validators](#built-in-validators)

> Inspired the projects [albrow/forms](https://github.com/albrow/forms) and [asaskevich/govalidator](https://github.com/asaskevich/govalidator) and [inhere/php-validate](https://github.com/inhere/php-validate). Thank you very much

## [中文说明](README.zh-CN.md)

中文说明请查看 **[README.zh-CN](README.zh-CN.md)**

## Go Doc

- [godoc for gopkg](https://pkg.go.dev/gopkg.in/gookit/validate.v1)
- [godoc for github](https://pkg.go.dev/github.com/gookit/validate)

## Validate Struct

With the `validate` tag of the structure, you can quickly verify a structure data.

provides extended functionality:

The struct can implement three interfaces methods, which is convenient to do some customization:

- `ConfigValidation(v *Validation)` will be called after the validator instance is created
- `Messages() map[string]string` can customize the validator error message
- `Translates() map[string]string` can customize field translation

**`v1.2.1` Update**:

- Support configuration field mapping through structure tag, read the value of `json` tag by default
- Support configuration error message via structure's `message` tag

```go
package main

import (
	"fmt"
	"time"

	"github.com/gookit/validate"
)

// UserForm struct
type UserForm struct {
	Name     string    `validate:"required|minLen:7"`
	Email    string    `validate:"email" message:"email is invalid"`
	Age      int       `validate:"required|int|min:1|max:99" message:"int:age must int| min: age min value is 1"`
	CreateAt int       `validate:"min:1"`
	Safe     int       `validate:"-"`
	UpdateAt time.Time `validate:"required"`
	Code     string    `validate:"customValidator"`
}

// CustomValidator custom validator in the source struct.
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
		fmt.Println(v.Errors.Field("Name")) // returns error messages of the field 
	}
}
```

## Validate Map

You can also validate a MAP data directly.

```go
package main

import (
"fmt"

"github.com/gookit/validate"
)

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
		safeData := v.SafeData()
		// do something ...
	} else {
		fmt.Println(v.Errors) // all error messages
		fmt.Println(v.Errors.One()) // returns a random error message text
	}
}
```

## Validate Request

If it is an HTTP request, you can quickly validate the data and pass the verification. Then bind the secure data to the structure.

```go
package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gookit/validate"
)

// UserForm struct
type UserForm struct {
	Name     string
	Email    string
	Age      int
	CreateAt int
	Safe     int
	UpdateAt time.Time
	Code     string
}

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
		v.FieldRule("code", `required|regex:\d{4,6}`)

		if v.Validate() { // validate ok
			// safeData := v.SafeData()
			userForm := &UserForm{}
			v.BindSafeData(userForm)

			// do something ...
			fmt.Println(userForm.Name)
		} else {
			fmt.Println(v.Errors) // all error messages
			fmt.Println(v.Errors.One()) // returns a random error message text
		}
	})

	http.ListenAndServe(":8090", handler)
}
```

## Quick Method

Quick create `Validation` instance.

- `New(data interface{}, scene ...string) *Validation`
- `Request(r *http.Request) *Validation`
- `JSON(s string, scene ...string) *Validation`
- `Struct(s interface{}, scene ...string) *Validation`
- `Map(m map[string]interface{}, scene ...string) *Validation`

Quick create `DataFace` instance.

- `FromMap(m map[string]interface{}) *MapData`
- `FromStruct(s interface{}) (*StructData, error)`
- `FromJSON(s string) (*MapData, error)`
- `FromJSONBytes(bs []byte) (*MapData, error)`
- `FromURLValues(values url.Values) *FormData`
- `FromRequest(r *http.Request, maxMemoryLimit ...int64) (DataFace, error)`

> Create `Validation` by `DataFace`

```go
d := FromMap(map[string]interface{}{"key": "val"})
v := d.Validation()
```

## More Usage

### Global Option

You can adjust some processing logic of the validator by changing the global option settings.

```go
// GlobalOption settings for validate
type GlobalOption struct {
	// FilterTag name in the struct tags. default: filter
	FilterTag string
	// ValidateTag in the struct tags. default: validate
	ValidateTag string
	// StopOnError If true: An error occurs, it will cease to continue to verify. default: true
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty. default: true
	SkipOnEmpty bool
}
```

Usage:

```go
// change global opts
validate.Config(func(opt *validate.GlobalOption) {
	opt.StopOnError = false
	opt.SkipOnEmpty = false
})
```

### Custom Error Messages

- Register language messages

```go
import "github.com/gookit/validate/locales/zhcn"

// for all Validation.
// NOTICE: must be register before on validate.New()
zhcn.RegisterGlobal()

v := validate.New()

// only for current Validation
zhcn.Register(v)
```

- Manual add global messages

```go
validate.AddGlobalMessages(map[string]string{
    "minLength": "OO! {field} min length is %d",
})
```

- Add messages for current validation

```go
v := validate.New(map[string]interface{}{
    "name": "inhere",
})
v.StringRule("name", "required|string|minLen:7|maxLen:15")

v.AddMessages(map[string]string{
    "minLength": "OO! {field} min length is %d",
    "name.minLen": "OO! username min length is %d",
})
```

- For a struct

```go
// Messages you can custom validator error messages. 
func (f UserForm) Messages() map[string]string {
	return validate.MS{
		"required": "oh! the {field} is required",
		"Name.required": "message for special field",
	}
}
```

### Add Custom Validator

`validate` supports adding custom validators, and supports adding `global validator` and `temporary validator`.

- **Global Validator** is globally valid and can be used everywhere
- **Temporary Validator** added to the current validation instance, only the current validation is available
- Add verification method to the structure. How to use please see the structure verification example above

> Note: The validator method must return a `bool` to indicate whether the validation was successful.
> The first parameter is the corresponding field value. If there are additional parameters, they will be appended automatically.

#### Add Global Validator

You can add one or more custom validators at once.

```go
validate.AddValidator("myCheck0", func(val interface{}) bool {
	// do validate val ...
	return true
})
validate.AddValidators(M{
	"myCheck1": func(val interface{}) bool {
		// do validate val ...
		return true
	},
})
```

#### Add Temporary Validator

Again, you can add one or more custom validators at once.

```go
v := validate.Struct(u)
v.AddValidator("myFunc3", func(val interface{}) bool {
	// do validate val ...
	return true
})
v.AddValidators(M{
	"myFunc4": func(val interface{}) bool {
		// do validate val ...
		return true
	},
})
```

## Use on gin framework

```go
package main
import (
    "github.com/gin-gonic/gin/binding"
    "github.com/gookit/validate"
)

// implements the binding.StructValidator
type customValidator struct {}

func (c *customValidator) ValidateStruct(ptr interface{}) error {
    v := validate.Struct(ptr)
    v.Validate() // do validating
    
    if v.Errors.Empty() {
	return nil
    }

    return v.Errors
}

func (c *customValidator) Engine() interface{} {
    return nil
}

func main()  {
	// ...

    // after init gin, set custom validator
    binding.Validator = &customValidator{}
}
```

<a id="built-in-validators"></a>
## Built In Validators

> Camel-style validator names now have underlined aliases. `endsWith` can also be written as `ends_with`

validator/aliases | description
-------------------|-------------------------------------------
`required`  | Check value is required and cannot be empty. 
`required_if/requiredIf`  | `required_if:anotherfield,value,...` The field under validation must be present and not empty if the `anotherField` field is equal to any value.
`requiredUnless`  | `required_unless:anotherfield,value,...` The field under validation must be present and not empty unless the `anotherField` field is equal to any value. 
`requiredWith`  | `required_with:foo,bar,...` The field under validation must be present and not empty only if any of the other specified fields are present.
`requiredWithAll`  | `required_with_all:foo,bar,...` The field under validation must be present and not empty only if all of the other specified fields are present.
`requiredWithout`  | `required_without:foo,bar,...` The field under validation must be present and not empty only when any of the other specified fields are not present.
`requiredWithoutAll`  | `required_without_all:foo,bar,...` The field under validation must be present and not empty only when all of the other specified fields are not present. 
`-/safe`  | The field values are safe and do not require validation
`int/integer/isInt`  | Check value is `intX` `uintX` type, And support size checking. eg: `"int"` `"int:2"` `"int:2,12"`
`uint/isUint`  |  Check value is uint(`uintX`) type, `value >= 0`
`bool/isBool`  |  Check value is bool string(`true`: "1", "on", "yes", "true", `false`: "0", "off", "no", "false").
`string/isString`  |  Check value is string type.
`float/isFloat`  |  Check value is float(`floatX`) type
`slice/isSlice`  |  Check value is slice type(`[]intX` `[]uintX` `[]byte` `[]string` ...).
`in/enum`  |  Check if the value is in the given enumeration
`not_in/notIn`  |  Check if the value is not in the given enumeration
`contains`  |  Check if the input value contains the given value
`not_contains/notContains`  |  Check if the input value not contains the given value
`string_contains/stringContains`  |  Check if the input string value is contains the given sub-string
`starts_with/startsWith`  |  Check if the input string value is starts with the given sub-string
`ends_with/endsWith`  |  Check if the input string value is ends with the given sub-string
`range/between`  |  Check that the value is a number and is within the given range
`max/lte`  |  Check value is less than or equal to the given value
`min/gte`  |  Check value is greater than or equal to the given value(for `intX` `uintX` `floatX`)
`eq/equal/isEqual`  |  Check that the input value is equal to the given value
`ne/notEq/notEqual`  |  Check that the input value is not equal to the given value
`lt/lessThan`  |  Check value is less than the given value(use for `intX` `uintX` `floatX`)
`gt/greaterThan`  |  Check value is greater than the given value(use for `intX` `uintX` `floatX`)
`email/isEmail`  |   Check value is email address string.
`intEq/intEqual`  |  Check value is int and equals to the given value.
`len/length`  |  Check value length is equals to the given size(use for `string` `array` `slice` `map`).
`regex/regexp`  |  Check if the value can pass the regular verification
`arr/array/isArray`  |   Check value is array type
`map/isMap`  |  Check value is a MAP type
`strings/isStrings`  |  Check value is string slice type(only allow `[]string`).
`ints/isInts`  |  Check value is int slice type(only allow `[]int`).
`min_len/minLen/minLength`  |  Check the minimum length of the value is the given size
`max_len/maxLen/maxLength`  |  Check the maximum length of the value is the given size
`eq_field/eqField`  |  Check that the field value is equals to the value of another field
`ne_field/neField`  |  Check that the field value is not equals to the value of another field
`gte_field/gteField`  |  Check that the field value is greater than or equal to the value of another field
`gt_field/gtField`  |  Check that the field value is greater than the value of another field
`lte_field/lteField`  |  Check if the field value is less than or equal to the value of another field
`lt_field/ltField`  |  Check that the field value is less than the value of another field
`file/isFile`  |  Verify if it is an uploaded file
`image/isImage`  |  Check if it is an uploaded image file and support suffix check
`mime/mimeType/inMimeTypes`  |  Check that it is an uploaded file and is in the specified MIME type
`date/isDate` | Check the field value is date string. eg `2018-10-25`
`gt_date/gtDate/afterDate` | Check that the input value is greater than the given date string.
`lt_date/ltDate/beforeDate` | Check that the input value is less than the given date string
`gte_date/gteDate/afterOrEqualDate` | Check that the input value is greater than or equal to the given date string.
`lte_date/lteDate/beforeOrEqualDate` | Check that the input value is less than or equal to the given date string.
`has_whitespace/hasWhitespace` | Check value string has Whitespace.
`ascii/ASCII/isASCII` | Check value is ASCII string.
`alpha/isAlpha` | Verify that the value contains only alphabetic characters
`alphaNum/isAlphaNum` | Check that only letters, numbers are included
`alphaDash/isAlphaDash` | Check to include only letters, numbers, dashes ( - ), and underscores ( _ )
`multiByte/isMultiByte` | Check value is MultiByte string.
`base64/isBase64` | Check value is Base64 string.
`dns_name/dnsName/DNSName/isDNSName` | Check value is DNSName string.
`data_uri/dataURI/isDataURI` | Check value is DataURI string.
`empty/isEmpty` | Check value is Empty string.
`hex_color/hexColor/isHexColor` | Check value is Hex color string.
`hexadecimal/isHexadecimal` | Check value is Hexadecimal string.
`json/JSON/isJSON` | Check value is JSON string.
`lat/latitude/isLatitude` | Check value is Latitude string.
`lon/longitude/isLongitude` | Check value is Longitude string.
`mac/isMAC` | Check value is MAC string.
`num/number/isNumber` | Check value is number string. `>= 0`
`cn_mobile/cnMobile/isCnMobile` | Check value is china mobile number string.
`printableASCII/isPrintableASCII` | Check value is PrintableASCII string.
`rgb_color/rgbColor/RGBColor/isRGBColor` | Check value is RGB color string.
`url/isURL` | Check value is URL string.
`fullUrl/isFullURL` | Check value is full URL string(_must start with http,https_).
`ip/isIP`  |  Check value is IP(v4 or v6) string.
`ipv4/isIPv4`  |  Check value is IPv4 string.
`ipv6/isIPv6`  |  Check value is IPv6 string.
`CIDR/isCIDR` | Check value is CIDR string.
`CIDRv4/isCIDRv4` | Check value is CIDRv4 string.
`CIDRv6/isCIDRv6` | Check value is CIDRv6 string.
`uuid/isUUID` | Check value is UUID string.
`uuid3/isUUID3` | Check value is UUID3 string.
`uuid4/isUUID4` | Check value is UUID4 string.
`uuid5/isUUID5` | Check value is UUID5 string.
`filePath/isFilePath` | Check value is an existing file path
`unixPath/isUnixPath` | Check value is Unix Path string.
`winPath/isWinPath` | Check value is Windows Path string.
`isbn10/ISBN10/isISBN10` | Check value is ISBN10 string.
`isbn13/ISBN13/isISBN13` | Check value is ISBN13 string.

**Notice:**

- `intX` is contains: int, int8, int16, int32, int64
- `uintX` is contains: uint, uint8, uint16, uint32, uint64
- `floatX` is contains: float32, float64

<a id="built-in-filters"></a>
## Built In Filters

> Filters powered by: [gookit/filter](https://github.com/gookit/filter)

filter/aliases | description 
-------------------|-------------------------------------------
`int`  | Convert value(string/intX/floatX) to `int` type `v.FilterRule("id", "int")`
`uint`  | Convert value(string/intX/floatX) to `uint` type `v.FilterRule("id", "uint")`
`int64`  | Convert value(string/intX/floatX) to `int64` type `v.FilterRule("id", "int64")`
`float`  | Convert value(string/intX/floatX) to `float` type
`bool`  | Convert string value to bool. (`true`: "1", "on", "yes", "true", `false`: "0", "off", "no", "false")
`trim/trimSpace`  | Clean up whitespace characters on both sides of the string
`ltrim/trimLeft`  | Clean up whitespace characters on left sides of the string
`rtrim/trimRight`  | Clean up whitespace characters on right sides of the string
`int/integer`  | Convert value(string/intX/floatX) to int type `v.FilterRule("id", "int")`
`lower/lowercase` | Convert string to lowercase
`upper/uppercase` | Convert string to uppercase
`lcFirst/lowerFirst` | Convert the first character of a string to lowercase
`ucFirst/upperFirst` | Convert the first character of a string to uppercase
`ucWord/upperWord` | Convert the first character of each word to uppercase
`camel/camelCase` | Convert string to camel naming style
`snake/snakeCase` | Convert string to snake naming style
`escapeJs/escapeJS` | Escape JS string.
`escapeHtml/escapeHTML` | Escape HTML string.
`str2ints/strToInts` | Convert string to int slice `[]int` 
`str2time/strToTime` | Convert date string to `time.Time`.
`str2arr/str2array/strToArray` | Convert string to string slice `[]string`

## Gookit packages

- [gookit/ini](https://github.com/gookit/ini) Go config management, use INI files
- [gookit/rux](https://github.com/gookit/rux) Simple and fast request router for golang HTTP 
- [gookit/gcli](https://github.com/gookit/gcli) build CLI application, tool library, running CLI commands
- [gookit/event](https://github.com/gookit/event) Lightweight event manager and dispatcher implements by Go
- [gookit/cache](https://github.com/gookit/cache) Generic cache use and cache manager for golang. support File, Memory, Redis, Memcached.
- [gookit/config](https://github.com/gookit/config) Go config management. support JSON, YAML, TOML, INI, HCL, ENV and Flags
- [gookit/color](https://github.com/gookit/color) A command-line color library with true color support, universal API methods and Windows support
- [gookit/filter](https://github.com/gookit/filter) Provide filtering, sanitizing, and conversion of golang data
- [gookit/validate](https://github.com/gookit/validate) Use for data validation and filtering. support Map, Struct, Form data
- [gookit/goutil](https://github.com/gookit/goutil) Some utils for the Go: string, array/slice, map, format, cli, env, filesystem, test and more
- More please see https://github.com/gookit

## See also

- https://github.com/albrow/forms
- https://github.com/asaskevich/govalidator
- https://github.com/go-playground/validator
- https://github.com/inhere/php-validate

## License

**[MIT](LICENSE)**
