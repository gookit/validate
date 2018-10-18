# Validate

[![GoDoc](https://godoc.org/github.com/gookit/validate?status.svg)](https://godoc.org/github.com/gookit/validate)
[![Build Status](https://travis-ci.org/gookit/validate.svg?branch=master)](https://travis-ci.org/gookit/validate)
[![Coverage Status](https://coveralls.io/repos/github/gookit/validate/badge.svg?branch=master)](https://coveralls.io/github/gookit/validate?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/validate)](https://goreportcard.com/report/github.com/gookit/validate)

Go通用的数据验证与过滤库，使用简单，内置大部分常用验证器、过滤器，支持自定义消息、字段翻译。

- 支持验证Map，Struct，Request（Form，JSON，url.Values）数据
- 简单方便，支持前置验证检查, 支持添加自定义验证器
- 支持将规则按场景进行分组设置。不同场景验证不同的字段
- 支持自定义每个验证的错误消息，字段翻译，消息翻译(内置`en` `zh-CN`)
- 支持在进行验证前对值使用过滤器进行净化过滤，查看 [内置过滤器](#built-in-filters)
- 方便的获取错误信息，验证后的安全数据获取(只会收集有规则检查过的数据)
- 已经内置了超多（> 60 个）常用的验证器，查看 [内置验证器](#built-in-validators)
- 完善的单元测试，覆盖率 > 85%

> 受到这些项目的启发 [albrow/forms](https://github.com/albrow/forms) 和 [asaskevich/govalidator](https://github.com/asaskevich/govalidator). 非常感谢它们

## Go Doc

- [godoc for gopkg](https://godoc.org/gopkg.in/gookit/validate.v1)
- [godoc for github](https://godoc.org/github.com/gookit/validate)

## 验证结构体(Struct)

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
	Code     string    `validate:"customValidator"` // 使用自定义验证器
}

// CustomValidator 定义在结构体中的自定义验证器
func (f UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}

// Messages 您可以自定义验证器错误消息
func (f UserForm) Messages() map[string]string {
	return validate.MS{
		"required": "oh! the {field} is required",
		"Name.required": "message for special field",
	}
}

// Translates 你可以自定义字段翻译
func (f UserForm) Translates() map[string]string {
	return validate.MS{
		"Name": "用户名称",
		"Email": "用户Email",
	}
}

func main() {
	u := &UserForm{
		Name: "inhere",
	}
	
	// 创建 Validation 实例
	v := validate.Struct(u)
	// v := validate.New(u)

	if v.Validate() { // 验证成功
		// do something ...
	} else {
		fmt.Println(v.Errors) // 所有的错误消息
		fmt.Println(v.Errors.One()) // 返回随机一条错误消息
		fmt.Println(v.Errors.Field("Name")) // 返回该字段的错误消息
	}
}
```

## 验证`Map`数据

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
	
	// 也可以这样，一次添加多个验证器
	v.StringRule("age", "required|int|min:1|max:99")
	v.StringRule("name", "required|minLen:7")

    // 设置不同场景验证不同的字段
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

## 验证请求

传入 `*http.Request`，快捷方法 `FromRequest()` 就会自动根据请求方法和请求数据类型收集相应的数据

- `GET/DELETE/...` 等，会搜集 url query 数据
- `POST/PUT/PATCH` 并且类型为 `application/json` 会搜集JSON数据
- `POST/PUT/PATCH` 并且类型为 `multipart/form-data` 会搜集表单数据，同时回收集文件上传数据
- `POST/PUT/PATCH` 并且类型为 `application/x-www-form-urlencoded` 会搜集表单数据

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

## 常用方法

快速创建 `Validation` 实例：

- `Request(r *http.Request) *Validation`
- `JSON(s string, scene ...string) *Validation`
- `Struct(s interface{}, scene ...string) *Validation`
- `Map(m map[string]interface{}, scene ...string) *Validation`
- `New(data interface{}, scene ...string) *Validation` 

快速创建 `DataFace` 实例：

- `FromMap(m map[string]interface{}) *MapData`
- `FromStruct(s interface{}) (*StructData, error)`
- `FromJSON(s string) (*MapData, error)`
- `FromJSONBytes(bs []byte) (*MapData, error)`
- `FromURLValues(values url.Values) *FormData`
- `FromRequest(r *http.Request, maxMemoryLimit ...int64) (DataFace, error)`

> 通过 `DataFace` 创建 `Validation` 

```go
d := FromMap(map[string]interface{}{"key": "val"})
v := d.Validation()
```

`Validation` 常用方法：

- Validation.`AtScene(scene string) *Validation` 设置当前验证场景名
- Validation.`Filtering() bool` 应用所有过滤规则
- Validation.`Validate() bool` 应用所有验证和过滤规则
- Validation.`SafeData() map[string]interface{}` 获取所有经过验证的数据

**提示**

- `intX` 包含: `int`, `int8`, `int16`, `int32`, `int64`
- `uintX` 包含: `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `floatX` 包含: `float32`, `float64`

<a id="built-in-filters"></a>
## 内置过滤器

> Filters powered by: [gookit/filter](https://github.com/gookit/filter)

过滤器/别名 | 描述信息 
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

<a id="built-in-validators"></a>
## 内置验证器

几大类别：

- 类型验证
- 字符串检查验证
- 大小、长度验证
- 字段值比较验证
- 上传文件验证
- 日期验证
- 其他验证

验证器/别名 | 描述信息
-------------------|-------------------------------------------
`required`  | 字段为必填项，值不能为空 
`-/safe`  | 标记当前字段是安全的，无需验证
`int/integer/isInt`  | 检查值是 `intX` `uintX` 类型
`uint/isUint`  |  检查值是 `uintX` 类型（`value >= 0`）
`bool/isBool`  |  检查值是布尔字符串(`true`: "1", "on", "yes", "true", `false`: "0", "off", "no", "false").
`string/isString`  |  检查值是字符串类型.
`float/isFloat`  |  检查值是 float(`floatX`) 类型
`slice/isSlice`  |  检查值是 slice 类型(`[]intX` `[]uintX` `[]byte` `[]string` 等).
`in/enum`  |  检查值是否在给定的枚举列表中
`notIn`  |  检查值不是在给定的枚举列表中
`contains`  |  检查输入值是否包含给定的值
`notContains`  |  检查输入值是否不包含给定值
`range/between`  |  检查值是否为数字且在给定范围内
`max/lte`  |  检查输入值小于或等于给定值
`min/gte`  |  检查输入值大于或等于给定值(for `intX` `uintX` `floatX`)
`intStr/intString/isIntString`  |  Check value is an int string.
`eq/equal/isEqual`  |  检查输入值是否等于给定值
`ne/notEq/notEqual`  |  检查输入值是否不等于给定值
`lt/lessThan`  |  Check value is less than the given size(use for `intX` `uintX` `floatX`)
`gt/greaterThan`  |  Check value is greater than the given size(use for `intX` `uintX` `floatX`)
`email/isEmail`  |   Check value is email address string.
`intEq/intEqual`  |  Check value is int and equals to the given value.
`len/length`  |  Check value length is equals to the given size(use for `string` `array` `slice` `map`).
`regex/regexp`  |  Check if the value can pass the regular verification
`arr/array/isArray`  |   Check value is array type
`map/isMap`  |  Check value is a MAP type
`strings/isStrings`  |  Check value is string slice type(only allow `[]string`).
`ints/isInts`  |  Check value is int slice type(only allow `[]int`).
`minLen/minLength`  |  Check the minimum length of the value is the given size
`maxLen/maxLength`  |  Check the maximum length of the value is the given size
`eqField`  |  Check that the field value is equals to the value of another field
`neField`  |  Check that the field value is not equals to the value of another field
`gteField`  |  Check that the field value is greater than or equal to the value of another field
`gtField`  |  Check that the field value is greater than the value of another field
`lteField`  |  Check if the field value is less than or equal to the value of another field
`ltField`  |  Check that the field value is less than the value of another field
`file/isFile`  |  验证是否是上传的文件
`image/isImage`  |  验证是否是上传的图片文件，支持后缀检查
`mime/mimeType/inMimeTypes`  |  验证是否是上传的文件，并且在指定的MIME类型中
`date/isDate` | Check the field value is date string. eg `2018-10-25`
`gtDate/afterDate` | Check that the input value is greater than the given date string.
`ltDate/beforeDate` | Check that the input value is less than the given date string
`gteDate/afterOrEqualDate` | Check that the input value is greater than or equal to the given date string.
`lteDate/beforeOrEqualDate` | Check that the input value is less than or equal to the given date string.
`hasWhitespace` | Check value string has Whitespace.
`ascii/ASCII/isASCII` | Check value is ASCII string.
`alpha/isAlpha` | 验证值是否仅包含字母字符
`alphaNum/isAlphaNum` | 验证是否仅包含字母、数字
`alphaDash/isAlphaDash` | 验证是否仅包含字母、数字、破折号（ - ）以及下划线（ _ ）
`multiByte/isMultiByte` | Check value is MultiByte string.
`base64/isBase64` | Check value is Base64 string.
`dnsName/DNSName/isDNSName` | Check value is DNSName string.
`dataURI/isDataURI` | Check value is DataURI string.
`empty/isEmpty` | Check value is Empty string.
`hexColor/isHexColor` | Check value is HexColor string.
`hexadecimal/isHexadecimal` | Check value is Hexadecimal string.
`json/JSON/isJSON` | Check value is JSON string.
`lat/latitude/isLatitude` | Check value is Latitude string.
`lon/longitude/isLongitude` | Check value is Longitude string.
`mac/isMAC` | Check value is MAC string.
`num/number/isNumber` | Check value is number string. `>= 0`
`printableASCII/isPrintableASCII` | Check value is PrintableASCII string.
`rgbColor/RGBColor/isRGBColor` | Check value is RGBColor string.
`url/isURL` | Check value is URL string.
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
`filePath/isFilePath` | Check value is FilePath string.
`unixPath/isUnixPath` | Check value is UnixPath string.
`winPath/isWinPath` | Check value is WinPath string.
`isbn10/ISBN10/isISBN10` | Check value is ISBN10 string.
`isbn13/ISBN13/isISBN13` | Check value is ISBN13 string.

## 参考项目

- https://github.com/albrow/forms
- https://github.com/asaskevich/govalidator
- https://github.com/go-playground/validator
- https://github.com/inhere/php-validate

## License

**[MIT](LICENSE)**