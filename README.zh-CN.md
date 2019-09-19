# Validate

[![GoDoc](https://godoc.org/github.com/gookit/validate?status.svg)](https://godoc.org/github.com/gookit/validate)
[![Build Status](https://travis-ci.org/gookit/validate.svg?branch=master)](https://travis-ci.org/gookit/validate)
[![Coverage Status](https://coveralls.io/repos/github/gookit/validate/badge.svg?branch=master)](https://coveralls.io/github/gookit/validate?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/validate)](https://goreportcard.com/report/github.com/gookit/validate)

Go通用的数据验证与过滤库，使用简单，内置大部分常用验证器、过滤器，支持自定义消息、字段翻译。

> **[EN README](README.md)**

- 支持验证Map，Struct，Request（Form，JSON，url.Values, UploadedFile）数据
- 简单方便，支持前置验证检查, 支持添加自定义验证器
- 支持将规则按场景进行分组设置，不同场景验证不同的字段
- 支持在进行验证前对值使用过滤器进行净化过滤，查看 [内置过滤器](#built-in-filters)
- 已经内置了超多（> 60 个）常用的验证器，查看 [内置验证器](#built-in-validators)
- 方便的获取错误信息，验证后的安全数据获取(只会收集有规则检查过的数据)
- 支持自定义每个验证的错误消息，字段翻译，消息翻译(内置`en` `zh-CN`)
- 完善的单元测试，测试覆盖率 > 90%

> 受到 [albrow/forms](https://github.com/albrow/forms) 和 [asaskevich/govalidator](https://github.com/asaskevich/govalidator) 这些项目的启发. 非常感谢它们

## Go Doc

- [godoc for gopkg](https://godoc.org/gopkg.in/gookit/validate.v1)
- [godoc for github](https://godoc.org/github.com/gookit/validate)

## 验证结构体(Struct)

结构体可以实现3个接口方法，方便做一些自定义：

- `ConfigValidation(v *Validation)` 将在创建验证器实例后调用
- `Messages() map[string]string` 可以自定义验证器错误消息
- `Translates() map[string]string` 可以自定义字段翻译

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
- `POST/PUT/PATCH` 并且类型为 `multipart/form-data` 会搜集表单数据，同时会收集文件上传数据
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

- `func (v *Validation) AtScene(scene string) *Validation` 设置当前验证场景名
- `func (v *Validation) Filtering() bool` 应用所有过滤规则
- `func (v *Validation) Validate() bool` 应用所有验证和过滤规则
- `func (v *Validation) SafeData() map[string]interface{}` 获取所有经过验证的数据
- `func (v *Validation) BindSafeData(ptr interface{}) error` 将验证后的安全数据绑定到一个结构体

## 更多使用

### 全局选项

```go
// GlobalOption settings for validate
type GlobalOption struct {
	// FilterTag name in the struct tags.
	FilterTag string
	// ValidateTag in the struct tags.
	ValidateTag string
	// StopOnError If true: An error occurs, it will cease to continue to verify
	StopOnError bool
	// SkipOnEmpty Skip check on field not exist or value is empty
	SkipOnEmpty bool
}
```

如何配置:

```go
	// change global opts
	validate.Config(func(opt *validate.GlobalOption) {
		opt.StopOnError = false
		opt.SkipOnEmpty = false
	})
```

### 自定义验证器

`validate` 支持添加自定义验证器，并且支持添加 `全局验证器` 和 `临时验证器` 两种

- **全局验证器** 全局有效，所有地方都可以使用
- **临时验证器** 添加到当前验证实例上，仅当次验证可用

#### 添加全局验证器

你可以一次添加一个或者多个自定义验证器

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

#### 添加临时验证器

同样，你可以一次添加一个或者多个自定义验证器

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
- 大小、长度验证
- 字段值比较验证
- 上传文件验证
- 日期验证
- 字符串检查验证
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
`eq/equal/isEqual`  |  检查输入值是否等于给定值
`ne/notEq/notEqual`  |  检查输入值是否不等于给定值
`lt/lessThan`  |  检查值小于给定大小(use for `intX` `uintX` `floatX`)
`gt/greaterThan`  |  检查值大于给定大小(use for `intX` `uintX` `floatX`)
`intEq/intEqual`  |  检查值为int且等于给定值
`len/length`  |  检查值长度等于给定大小(use for `string` `array` `slice` `map`).
`minLen/minLength`  |  检查值的最小长度是给定大小
`maxLen/maxLength`  |  检查值的最大长度是给定大小
`email/isEmail`  |   检查值是Email地址字符串
`regex/regexp`  |  检查该值是否可以通过正则验证
`arr/array/isArray`  |  检查值是数组`array`类型
`map/isMap`  |  检查值是MAP类型
`strings/isStrings`  |  检查值是字符串切片类型(`[]string`).
`ints/isInts`  |  检查值是int slice类型(only allow `[]int`).
`eqField`  |  检查字段值是否等于另一个字段的值
`neField`  |  检查字段值是否不等于另一个字段的值
`gtField`  |  检查字段值是否大于另一个字段的值
`gteField`  | 检查字段值是否大于或等于另一个字段的值
`ltField`  |  检查字段值是否小于另一个字段的值
`lteField`  |  检查字段值是否小于或等于另一个字段的值
`file/isFile`  |  验证是否是上传的文件
`image/isImage`  |  验证是否是上传的图片文件，支持后缀检查
`mime/mimeType/inMimeTypes`  |  验证是否是上传的文件，并且在指定的MIME类型中
`date/isDate` | 检查字段值是否为日期字符串。（只支持几种常用的格式） eg `2018-10-25`
`gtDate/afterDate` | 检查输入值是否大于给定的日期字符串
`ltDate/beforeDate` | 检查输入值是否小于给定的日期字符串
`gteDate/afterOrEqualDate` | 检查输入值是否大于或等于给定的日期字符串
`lteDate/beforeOrEqualDate` | 检查输入值是否小于或等于给定的日期字符串
`hasWhitespace` | 检查字符串值是否有空格
`ascii/ASCII/isASCII` | 检查值是ASCII字符串
`alpha/isAlpha` | 验证值是否仅包含字母字符
`alphaNum/isAlphaNum` | 验证是否仅包含字母、数字
`alphaDash/isAlphaDash` | 验证是否仅包含字母、数字、破折号（ - ）以及下划线（ _ ）
`multiByte/isMultiByte` | Check value is MultiByte string.
`base64/isBase64` | 检查值是Base64字符串
`dnsName/DNSName/isDNSName` | 检查值是DNS名称字符串
`dataURI/isDataURI` | Check value is DataURI string.
`empty/isEmpty` | Check value is Empty string.
`hexColor/isHexColor` | 检查值是16进制的颜色字符串
`hexadecimal/isHexadecimal` | 检查值是十六进制字符串
`json/JSON/isJSON` | 检查值是JSON字符串。
`lat/latitude/isLatitude` | 检查值是纬度坐标
`lon/longitude/isLongitude` | 检查值是经度坐标
`mac/isMAC` | 检查值是MAC字符串
`num/number/isNumber` | 检查值是数字字符串. `>= 0`
`printableASCII/isPrintableASCII` | Check value is PrintableASCII string.
`rgbColor/RGBColor/isRGBColor` | 检查值是RGB颜色字符串
`fullUrl/isFullURL` | 检查值是完整的URL字符串(_必须以http,https开始的URL_).
`url/isURL` | 检查值是URL字符串
`ip/isIP`  |  检查值是IP（v4或v6）字符串
`ipv4/isIPv4`  |  检查值是IPv4字符串
`ipv6/isIPv6`  |  检查值是IPv6字符串
`CIDR/isCIDR` | 检查值是 CIDR 字符串
`CIDRv4/isCIDRv4` | 检查值是 CIDR v4 字符串
`CIDRv6/isCIDRv6` | 检查值是 CIDR v6 字符串
`uuid/isUUID` | 检查值是UUID字符串
`uuid3/isUUID3` | 检查值是UUID3字符串
`uuid4/isUUID4` | 检查值是UUID4字符串
`uuid5/isUUID5` | 检查值是UUID5字符串
`filePath/isFilePath` | 检查值是一个存在的文件路径
`unixPath/isUnixPath` | 检查值是Unix Path字符串
`winPath/isWinPath` | 检查值是Windows路径字符串
`isbn10/ISBN10/isISBN10` | 检查值是ISBN10字符串
`isbn13/ISBN13/isISBN13` | 检查值是ISBN13字符串

**提示**

- `intX` 包含: `int`, `int8`, `int16`, `int32`, `int64`
- `uintX` 包含: `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `floatX` 包含: `float32`, `float64`

## 欢迎star

- **[github](https://github.com/gookit/validate)**
- [gitee](https://gitee.com/inhere/validate)

## Gookit 工具包

- [gookit/ini](https://github.com/gookit/ini) INI配置读取管理，支持多文件加载，数据覆盖合并, 解析ENV变量, 解析变量引用
- [gookit/rux](https://github.com/gookit/rux) Simple and fast request router for golang HTTP 
- [gookit/gcli](https://github.com/gookit/gcli) Go的命令行应用，工具库，运行CLI命令，支持命令行色彩，用户交互，进度显示，数据格式化显示
- [gookit/event](https://github.com/gookit/event) Go实现的轻量级的事件管理、调度程序库, 支持设置监听器的优先级, 支持对一组事件进行监听
- [gookit/cache](https://github.com/gookit/cache) 通用的缓存使用包装库，通过包装各种常用的驱动，来提供统一的使用API
- [gookit/config](https://github.com/gookit/config) Go应用配置管理，支持多种格式（JSON, YAML, TOML, INI, HCL, ENV, Flags），多文件加载，远程文件加载，数据合并
- [gookit/color](https://github.com/gookit/color) CLI 控制台颜色渲染工具库, 拥有简洁的使用API，支持16色，256色，RGB色彩渲染输出
- [gookit/filter](https://github.com/gookit/filter) 提供对Golang数据的过滤，净化，转换
- [gookit/validate](https://github.com/gookit/validate) Go通用的数据验证与过滤库，使用简单，内置大部分常用验证、过滤器
- [gookit/goutil](https://github.com/gookit/goutil) Go 的一些工具函数，格式化，特殊处理，常用信息获取等
- 更多请查看 https://github.com/gookit

## 参考项目

- https://github.com/albrow/forms
- https://github.com/asaskevich/govalidator
- https://github.com/go-playground/validator
- https://github.com/inhere/php-validate

## License

**[MIT](LICENSE)**
