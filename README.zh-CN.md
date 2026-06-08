# Validate

[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/gookit/validate)](https://github.com/gookit/validate)
[![GoDoc](https://pkg.go.dev/badge/github.com/gookit/validate.svg)](https://pkg.go.dev/github.com/gookit/validate)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/gookit/goutil?style=flat-square)
[![Coverage Status](https://coveralls.io/repos/github/gookit/validate/badge.svg?branch=master)](https://coveralls.io/github/gookit/validate?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gookit/validate)](https://goreportcard.com/report/github.com/gookit/validate)
[![Actions Status](https://github.com/gookit/validate/workflows/Unit-Tests/badge.svg)](https://github.com/gookit/validate/actions)

Go 通用的数据验证与过滤库，使用简单，内置大部分常用验证器、过滤器，支持自定义消息、字段翻译。

> **[README](README.md)**

- 简单方便，支持前置验证检查, 支持添加自定义验证器
  - 大多数过滤器和验证器都有别名方便使用
- 支持验证 `Map` `Struct` `Request`（`Form`，`JSON`，`url.Values`, `UploadedFile`）数据
  - 能简单快速的配置规则并验证 Map 数据
  - 能根据请求数据类型 `Content-Type` 快速验证 `http.Request` 并收集数据
  - 支持检查 slice 中的每个子项值. eg: `v.StringRule("tags.*", "required|string|minlen:1")`
- 支持将规则按场景进行分组设置，不同场景验证不同的字段
  - 已经内置了超多（**>70** 个）常用的验证器，查看 [内置验证器](#built-in-validators)
- 支持在进行验证前对值使用过滤器进行净化过滤，适应更多场景
  - 已经内置了超多（**>20** 个）常用的过滤器，查看 [内置过滤器](#built-in-filters)
- 方便的获取错误信息，验证后的安全数据获取(_只会收集有规则检查过的数据_)
- 支持自定义每个验证的错误消息，字段翻译，消息翻译(内置`en` `zh-CN` `zh-TW`)
  - 在结构体上可以使用 `message`, `label` 标签定义消息翻译
- 可以在任何框架中使用 `validate`，例如 Gin、Echo、Chi 等
- 支持直接使用规则来验证值 例如: `validate.Val("xyz@mail.com", "required|email")`
- 完善的单元测试，测试覆盖率 **> 90%**

## Go Doc

- [Godoc for github](https://pkg.go.dev/github.com/gookit/validate/v2)

> **v2.0**：模块路径已变更为 `github.com/gookit/validate/v2`（要求 Go 1.21+）。
> 从 v1.x 升级请参阅 [升级指南](docs/UPGRADE-v2.md)。
>
> **性能对比**：查看 [基准测试对比](docs/benchmark-v1-to-v2.md)，了解 v1.5.7 → v1.6.0 → v2.0.0 的性能演进。

安装：

```bash
go get github.com/gookit/validate/v2
```

## 验证结构体(Struct)

在结构体上添加 `validate`标签，可以快速对一个结构体进行验证设置。

### 使用标签快速配置验证

可以搭配使用 `message` 和 `label` 标签，快速配置结构体的字段翻译和错误消息。

- 支持通过结构体配置字段输出名称，默认读取 `json` 标签的值
- 支持通过结构体的 `message` tag 配置错误消息
- 支持通过结构体的 `label` tag 字段映射/翻译

**代码示例**:

```go
package main

import (
	"fmt"
	"time"

	"github.com/gookit/validate/v2"
)

// UserForm struct
type UserForm struct {
	Name     string    `validate:"required|min_len:7" message:"required:{field} is required" label:"用户名称"`
	Email    string    `validate:"email" message:"email is invalid" label:"用户邮箱"`
	Age      int       `validate:"required|int|min:1|max:99" message:"int:age must int|min:age min value is 1"`
	CreateAt int       `validate:"min:1"`
	Safe     int       `validate:"-"` // 标记字段安全无需验证
	UpdateAt time.Time `validate:"required" message:"update time is required"`
	Code     string    `validate:"customValidator"`
	// 结构体嵌套
	ExtInfo struct{
		Homepage string `validate:"required" label:"用户主页"`
		CityName string
	} `validate:"required" label:"扩展信息"`
}

// CustomValidator custom validator in the source struct.
func (f UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}
```

### 使用结构体方法配置验证

结构体可以实现3个接口方法，方便做一些自定义：

- `ConfigValidation(v *Validation)` 将在创建验证器实例后调用
- `Messages() map[string]string` 可以自定义验证器错误消息
- `Translates() map[string]string` 可以自定义字段映射/翻译

**代码示例**:

```go
package main

import (
	"fmt"
	"time"

	"github.com/gookit/validate/v2"
)

// UserForm struct
type UserForm struct {
	Name     string    `validate:"required|minLen:7"`
	Email    string    `validate:"email"`
	Age      int       `validate:"required|int|min:1|max:99"`
	Safe     int       `validate:"-"`
	CreateAt int       `validate:"min:1"`
	UpdateAt time.Time `validate:"required"`
	Code     string    `validate:"customValidator"` // 使用自定义验证器
	// 结构体嵌套
	ExtInfo  struct{
		Homepage string `validate:"required"`
		CityName string
    } `validate:"required"`
}

// CustomValidator 定义在结构体中的自定义验证器
func (f UserForm) CustomValidator(val string) bool {
	return len(val) == 4
}

// ConfigValidation 配置验证
// - 定义验证场景
// - 也可以添加验证设置
func (f UserForm) ConfigValidation(v *validate.Validation) {
	// v.StringRule()
	
	v.WithScenes(validate.SValues{
		"add":    []string{"ExtInfo.Homepage", "Name", "Code"},
		"update": []string{"ExtInfo.CityName", "Name"},
	})
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
		"Email": "用户邮箱",
		"ExtInfo.Homepage": "用户主页",
	}
}
```

### 创建和调用验证

```go
func main() {
	u := &UserForm{
		Name: "inhere",
	}
	
	// 创建 Validation 实例
	v := validate.Struct(u)
	// 或者使用
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

### 子结构体(嵌套)验证

默认情况下(`CheckSubOnParentMarked=true`),**具名**子结构体字段
(`struct` / `*struct` / slice-of-struct / map-of-struct)**只有带 `validate` tag 时才会被下探**收集其内部规则。
tag 的**值可为空** —— `validate:""` 即可作为标记;完全没有 `validate` tag 的具名字段**不会被下探**。
这是 Java `@Valid` 风格的「按需级联」,避免无意义地递归整棵结构体。

**匿名嵌入结构体豁免**:`type Bar struct { Foo }` 这种匿名嵌入(字段被提升、属父结构体的一部分)
**始终级联**,无需 tag;只有**具名**嵌套字段需要标记。

```go
type Address struct {
    City string `validate:"required"`
}
type User struct {
    Name string  `validate:"required"`
    Home Address `validate:"required"` // 有 tag → 下探,校验 Home.City
    Work Address `validate:""`         // 空 tag 也算标记 → 下探
    Temp Address                       // 无 tag → 不下探,Temp.City 被忽略
}
```

如需全局恢复 v1 的「无条件级联」行为:

```go
validate.Config(func(o *validate.GlobalOption) {
	o.CheckSubOnParentMarked = false
})
```

> 关于此行为变更的更多说明,请参阅[升级指南](docs/UPGRADE-v2.md)。

## 验证`Map`数据

```go
package main

import "fmt"
import "github.com/gookit/validate/v2"

func main()  {
	m := map[string]any{
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
	// feat: 支持在 slice 中检查子项的值
	v.StringRule("tags.*", "required|string|min_len:7")

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

import (
	"fmt"
	"net/http"

	"github.com/gookit/validate/v2"
)

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
		
		vr := v.ValidateR()
		if vr.IsOK() { // validate ok
			// safeData := vr.SafeData()
			userForm := &UserForm{}
			vr.BindSafeData(userForm)

			// do something ...
			fmt.Println(userForm.Name)
		} else {
			fmt.Println(vr.Errors) // all error messages
			fmt.Println(vr.Errors.One()) // returns a random error message text
		}
	})
	
	http.ListenAndServe(":8090", handler)
}
```

## 常用方法

快速校验 struct（内部池化，无需手动管理生命周期）：

- `Check(structPtr any, scene ...string) *ValidResult` —— struct 校验的推荐默认入口。调用方无状态、内部自动池化复用实例，返回携带清洗/安全数据与绑定方法的 `*ValidResult`。
- `CheckErr(structPtr any, scene ...string) error` —— **opt-in 快速过/败入口**，适用于"只要过/败、不取清洗数据、不绑定"的高频场景（如中间件快速 reject）。与 `Check` 一样池化，但**跳过收集 safeData/filteredData**、也不构建 `*ValidResult`，因此分配更少（struct 合法路径：3 allocs，对照 `Check` 6）。通过返回 `nil`，否则返回首个错误。

```go
// 快速过/败（如中间件 reject）—— 不需要清洗后的数据
if err := validate.CheckErr(&u); err != nil {
	return err
}

// 需要清洗数据或 BindStruct？改用 Check / ValidateR
r := validate.Check(&u)
if r.Fail() {
	return r.Err()
}
r.BindSafeData(&out)
```

> `CheckErr` 仅支持 struct 数据源。Map 数据或编程式规则请用 `New`/`Map` + `StringRule...` 构建 `Validation` 后调用 `ValidateErr()`。

快速创建 `Validation` 实例：

- `Request(r *http.Request) *Validation`
- `JSON(s string, scene ...string) *Validation`
- `Struct(s any, scene ...string) *Validation`
- `Map(m map[string]any, scene ...string) *Validation`
- `New(data any, scene ...string) *Validation` 

快速创建 `DataFace` 实例：

- `FromMap(m map[string]any) *MapData`
- `FromStruct(s any) (*StructData, error)`
- `FromJSON(s string) (*MapData, error)`
- `FromJSONBytes(bs []byte) (*MapData, error)`
- `FromURLValues(values url.Values) *FormData`
- `FromRequest(r *http.Request, maxMemoryLimit ...int64) (DataFace, error)`

> 通过 `DataFace` 创建 `Validation` 

```go
d := FromMap(map[string]any{"key": "val"})
v := d.Validation()
```

### `Validation` 常用方法：

- `func (v *Validation) AtScene(scene string) *Validation` 设置当前验证场景名
- `func (v *Validation) Filtering() bool` 应用所有过滤规则
- `func (v *Validation) Validate() bool` 应用所有验证和过滤规则，返回是否验证成功
- `func (v *Validation) ValidateE() Errors` 应用所有验证和过滤规则，并在失败时返回错误
- `func (v *Validation) ValidateR() *ValidResult` 验证并返回与实例解耦的结果对象 `ValidResult`

> **v2.0 起 `Validation` 仅保留「过/败 + 错误」面**：获取安全数据/绑定结构体的方法
> （`SafeData()` / `Safe()` / `SafeVal()` / `Filtered()` / `FilteredData()` / `BindSafeData()` / `BindStruct()`）
> 已移至结果对象 `ValidResult`。请通过 `validate.Check(&u)`（推荐，struct，内部池化）或
> `v.ValidateR()`（map/builder）拿到 `*ValidResult`，再调用：
> - `func (r *ValidResult) SafeData() map[string]any` 获取所有经过验证的安全数据
> - `func (r *ValidResult) BindSafeData(ptr any) error` 将安全数据绑定到结构体（别名 `BindStruct`）
>
> 只需要「过/败」结果时，用 `func validate.CheckErr(structPtr any, scene ...string) error`（最少分配）。

## 更多使用

### 验证错误信息

`v.Errors` 是一个Map数据，键是字段名，值是 `map[string]string`。

```go
// do validating
if v.Validate() {
	return nil
}

// get errors
es := v.Errors

// check
es.Empty() // bool

// 返回一个随机的 error错误，如果没有错误返回 nil
fmt.Println(v.Errors.OneError())
fmt.Println(v.Errors.ErrOrNil())

fmt.Println(v.Errors) // all error messages
fmt.Println(v.Errors.One()) // returns a random error message text
fmt.Println(v.Errors.Field("Name")) // returns error messages of the field 
```

**错误转为JSON**:

- `StopOnError=true`(默认)，只会有一个错误:

```json
{
    "field1": {
        "required": "error msg0"
    }
}
```

- `StopOnError=false`时，可能会返回多个错误:

```json
{
    "field1": {
        "minLen": "error msg1",
        "required": "error msg0"
    },
    "field2": {
        "min": "error msg2"
    }
}
```

### 全局选项

你可以通过改变全局选项设置，来调整验证器的一些处理逻辑。

```go
// GlobalOption settings for validate
type GlobalOption struct {
	// FilterTag 结构体中的过滤规则标签名称。默认 'filter`
	FilterTag string
	// ValidateTag 结构体中的验证规则标签名称。默认 'validate`
	ValidateTag string
	// FieldTag 定义结构体字段验证错误时的输出名字。默认使用 json
	FieldTag string
	// LabelTag 定义结构体字段验证错误时的输出翻译名称。默认使用 label
	// - 等同于设置 字段 translate
	LabelTag string
	// MessageTag define error message for the field. default: message
	MessageTag string
	// StopOnError 如果为 true，则出现第一个错误时，将停止继续验证。默认 true
	StopOnError bool
	// SkipOnEmpty 跳过对字段不存在或值为空的检查。默认 true
	SkipOnEmpty bool
	// UpdateSource Whether to update source field value, useful for struct validate
	UpdateSource bool
	// CheckDefault Whether to validate the default value set by the user
	CheckDefault bool
	// CheckZero Whether validate the default zero value. (intX,uintX: 0, string: "")
	CheckZero bool
	// CheckSubOnParentMarked 当字段是一个结构体时，仅在当前字段配置了验证tag时才收集子结构体的规则
	CheckSubOnParentMarked bool
}
```

**如何配置**:

```go
	// 更改全局选项
	validate.Config(func(opt *validate.GlobalOption) {
		opt.StopOnError = false
		opt.SkipOnEmpty = false
	})
```

### 自定义错误消息

- 注册内置的语言消息

```go
import "github.com/gookit/validate/v2/locales/zhcn"

// for all Validation.
// NOTICE: 必须在调用 validate.New() 前注册, 它只需要一次调用。
zhcn.RegisterGlobal()

// ... ...

v := validate.New()

// only for current Validation
zhcn.Register(v)
```

- 手动添加全局消息

```go
validate.AddGlobalMessages(map[string]string{
    "minLength": "OO! {field} min length is %d",
})
```

- 为当前验证添加消息(_仅本次验证有效_)

```go
v := validate.New(map[string]any{
    "name": "inhere",
})
v.StringRule("name", "required|string|minLen:7|maxLen:15")

v.AddMessages(map[string]string{
    "minLength": "OO! {field} min length is %d",
    "name.minLen": "OO! username min length is %d",
})
```

- 使用结构体标签: `message, label`

```go
type UserForm struct {
    Name     string    `validate:"required|minLen:7" label:"用户名称"`
    Email    string    `validate:"email" message:"email is invalid" label:"用户邮箱"`
}
```

- 结构体可以通过 `Messages()` 方法添加

```go
// Messages you can custom validator error messages. 
func (f UserForm) Messages() map[string]string {
	return validate.MS{
		"required": "oh! the {field} is required",
		"Name.required": "message for special field",
	}
}
```

### 自定义验证器

`validate` 支持添加自定义验证器，并且支持添加 `全局验证器` 和 `临时验证器` 两种

- **全局验证器** 全局有效，注册后所有地方都可以使用
- **临时验证器** 添加到当前验证实例上，仅当次验证可用
- 在结构体上添加验证方法。使用请看上面结构体验证示例中的 `func (f UserForm) CustomValidator(val string) bool`

> 注意：验证器方法必须返回一个 `bool` 表明验证是否成功。第一个参数是对应的字段值，如果有额外参数则自动追加在后面

#### 添加全局验证器

你可以一次添加一个或者多个自定义验证器

```go
	validate.AddValidator("myCheck0", func(val any) bool {
		// do validate val ...
		return true
	})
	validate.AddValidators(M{
		"myCheck1": func(val any) bool {
			// do validate val ...
			return true
		},
	})
```

#### 添加临时验证器

同样，你可以一次添加一个或者多个自定义验证器

```go
	v := validate.Struct(u)
	v.AddValidator("myFunc3", func(val any) bool {
		// do validate val ...
		return true
	})
	v.AddValidators(M{
		"myFunc4": func(val any) bool {
			// do validate val ...
			return true
		},
	})
```

### 添加自定义过滤器

`validate` 也允许添加自定义过滤器, 同样支持 `全局过滤器` 和 `临时过滤器` 两种

- **全局过滤器** 全局有效，注册后所有地方都可以使用
- **临时过滤器** 添加到当前验证实例上，仅当次验证可用

> TIP: 对于过滤器函数，允许具有 1 个返回值 `inteface{}` 或 2 个返回值 `(inteface{},error)` 的函数，其中第二个可以返回错误

#### 添加全局过滤器

你可以一次添加一个或者多个自定义过滤器

```go
package main

import "github.com/gookit/validate/v2"

func init() {
	validate.AddFilter("myToIntFilter0", func(val any) int {
		// do filtering val ...
		return 1
	})
	validate.AddFilters(validate.M{
		"myToIntFilter1": func(val any) (int, error) {
			// do filtering val ...
			return 1, nil
		},
	})
}
```

#### 添加临时过滤器

同样，你可以一次添加一个或者多个自定义过滤器

```go
package main

import "github.com/gookit/validate/v2"

func main() {
	v := validate.New(&someStrcut{})

	v.AddFilter("myToIntFilter0", func(val any) int {
		// do filtering val ...
		return 1
	})
	v.AddFilters(validate.M{
		"myToIntFilter1": func(val any) (int, error) {
			// do filtering val ...
			return 1, nil
		},
	})
}
```

### 自定义 `required` 验证器

允许自定义 `required` 验证器来自定义验证是否为空。但是，需注意验证器名称必须以 `required` 开头，例如 `required_custom`。

```go
	type Data struct {
		Age  int    `validate:"required_custom" message:"age is required"`
		Name string `validate:"required"`
	}

	v := validate.New(&Data{
		Name: "tom",
		Age:  0,
	})

	v.AddValidator("required_custom", func(val any) bool {
		// do check value
		return false
	})

	ok := v.Validate()
	assert.False(t, ok)
```

### 注册自定义类型(`AddCustomType`)

对于自定义/包装类型(如 `sql.NullString`),可以注册一个「底层值提取器」。提取后的值会
照常走现有的 `required` / 数值比较 / 长度 / 字符串校验路径,如同它本来就是个普通值。
对标 go-playground/validator 的 `RegisterCustomTypeFunc`。

- 签名 `AddCustomType(fn CustomTypeFunc, types ...any)`,其中
  `type CustomTypeFunc func(field reflect.Value) any`。
- 提取器返回 `nil` 表示「空/未设置」(此时 `required` 会失败)。
- 按样例的精确 `reflect.Type` 匹配,**不自动解指针**:传 `sql.NullString{}` 只匹配值类型;
  若要同时匹配 `*sql.NullString`,需再传入 `&sql.NullString{}` 样例。
- `ResetCustomTypes()` 清空注册表(测试/清理用)。
- 未注册任何自定义类型时,校验热路径零额外开销。

```go
import (
	"database/sql"
	"reflect"

	"github.com/gookit/validate/v2"
)

// 从 sql.NullString 提取底层字符串;无效则返回 nil(视为空)
validate.AddCustomType(func(field reflect.Value) any {
	ns := field.Interface().(sql.NullString)
	if !ns.Valid {
		return nil
	}
	return ns.String
}, sql.NullString{})

type Form struct {
	Name sql.NullString `validate:"required|minLen:3"`
}

v := validate.Struct(&Form{Name: sql.NullString{Valid: true, String: "inhere"}})
_ = v.Validate() // true:提取出 "inhere" 后按 required|minLen:3 校验
```

### 池化工厂(`NewFactory`)

当你需要大批量校验**同类型**数据时,可以用 opt-in 的 `Factory` 从**私有池**复用
`*Validation` 实例,摊销构造成本。`Struct` / `Map` / `New` 的默认行为与生命周期
**完全不变** —— 只有显式使用 `Factory` 的调用方才会走池化路径。

> 大多数场景直接用顶层 `validate.Check(&u)` 即可——它内部用包级池自动复用实例。
> 只有当你想要一个**私有**池时才需要 `Factory`。

```go
f := validate.NewFactory()
for i := range users {
	// ValidateR 会把结果快照进 ValidResult 并自动归还实例到池(无需手动 Release):
	r := f.Struct(&users[i]).ValidateR()
	if r.Fail() {
		// 使用 r.Errors / r.SafeData() / r.BindStruct(&out) ...
	}
}
```

若只需要布尔/错误面,可 `Validate()` 后 `Release()`:

```go
v := f.Struct(&users[i])
if !v.Validate() { /* 使用 v.Errors */ }
v.Release() // Reset 后归还到池; 之后不要再使用 v
```

## 在gin框架中使用

可以在任何框架中使用 `validate`，例如 Gin、Echo、Chi 等。 这里以 gin 示例:

```go
package main
import (
    "github.com/gin-gonic/gin/binding"
    "github.com/gookit/validate/v2"
)

// implements the binding.StructValidator
type customValidator struct {}

func (c *customValidator) ValidateStruct(ptr any) error {
    v := validate.Struct(ptr)
    v.Validate() // 调用验证

    return v.Errors
}

func (c *customValidator) Engine() any {
    return nil
}

func main()  {
	// ...

    // after init gin, set custom validator
    binding.Validator = &customValidator{}
}
```

<a id="built-in-validators"></a>
## 内置验证器

几大类别：

- (为空)必填验证
- 类型验证
- 大小、长度验证
- 字段值比较验证
- 文件验证
- 日期验证
- 字符串检查验证
- 其他验证

> TIP: 驼峰式的验证器名称现在都添加了下划线式的别名。因此 `endsWith` 也可以写为 `ends_with`

验证器/别名 | 描述信息
-------------------|-------------------------------------------
`required`  | 字段为必填项，值不能为空 
`required_if/requiredIf`  | `required_if:anotherfield,value,...` 如果其它字段 _anotherField_ 为任一值 _value_ ，则此验证字段必须存在且不为空。
`required_unless/requiredUnless`  | `required_unless:anotherfield,value,...` 如果其它字段 _anotherField_ 不等于任一值 _value_ ，则此验证字段必须存在且不为空。 
`required_with/requiredWith`  | `required_with:foo,bar,...` 在其他任一指定字段出现时，验证的字段才必须存在且不为空 
`required_with_all/requiredWithAll`  | `required_with_all:foo,bar,...` 只有在其他指定字段全部出现时，验证的字段才必须存在且不为空 
`required_without/requiredWithout`  | `required_without:foo,bar,...` 在其他指定任一字段不出现时，验证的字段才必须存在且不为空
`required_without_all/requiredWithoutAll`  | `required_without_all:foo,bar,...` 只有在其他指定字段全部不出现时，验证的字段才必须存在且不为空 
`-/safe`  | 标记当前字段是安全的，无需验证
`int/integer/isInt`  | 检查值是 `intX` `uintX` 类型，同时支持大小检查 `"int"` `"int:2"` `"int:2,12"`
`intStr/intString/isIntString`  | 检查值是整数数字字符串(允许前导符号 `+`/`-`)。例如 `"10"` `"-10"`
`uint/isUint`  |  检查值是 `uintX` 类型(`value >= 0`)
`bool/isBool`  |  检查值是布尔字符串(`true`: "1", "on", "yes", "true", `false`: "0", "off", "no", "false").
`string/isString`  |  检查值是字符串类型，同时支持长度检查 `"string"` `"string:2"` `"string:2,12"`
`float/isFloat`  |  检查值是 float(`floatX`) 类型
`slice/isSlice`  |  检查值是 slice 类型(`[]intX` `[]uintX` `[]byte` `[]string` 等).
`in/enum`  |  检查值()是否在给定的枚举列表(`[]string`, `[]intX`, `[]uintX`)中
`not_in/notIn`  |  检查值不是在给定的枚举列表中
`contains`  |  检查输入值(`string` `array/slice` `map`)是否包含给定的值
`not_contains/notContains`  |  检查输入值(`string` `array/slice` `map`)是否不包含给定值
`string_contains/stringContains`  |  检查输入的 `string` 值是否不包含给定sub-string值
`starts_with/startsWith`  |  检查输入的 `string` 值是否以给定sub-string开始
`ends_with/endsWith`  |  检查输入的 `string` 值是否以给定sub-string结束
`range/between`  |  检查值是否为数字且在给定范围内
`max/lte`  |  检查输入值小于或等于给定值
`min/gte`  |  检查输入值大于或等于给定值(for `intX` `uintX` `floatX`)
`eq/equal/isEqual`  |  检查输入值是否等于给定值
`ne/notEq/notEqual`  |  检查输入值是否不等于给定值
`lt/lessThan`  |  检查值小于给定大小(use for `intX` `uintX` `floatX`)
`gt/greaterThan`  |  检查值大于给定大小(use for `intX` `uintX` `floatX`)
`int_eq/intEq/intEqual`  |  检查值为int且等于给定值
`len/length`  |  检查值长度等于给定大小(use for `string` `array` `slice` `map`).
`min_len/minLen/minLength`  |  检查值的最小长度是给定大小
`max_len/maxLen/maxLength`  |  检查值的最大长度是给定大小
`strLen/strLength/stringLength`  |  检查字符串长度(按 rune/字符 计数)是否在给定范围。例如 `"strLength:3"` `"strLength:3,12"`
`email/isEmail`  |   检查值是Email地址字符串
`regex/regexp`  |  检查该值是否可以通过正则验证
`arr/list/array/isArray`  |  检查值是 `array` 或者 `slice`类型
`map/isMap`  |  检查值是 `map` 类型
`strings/isStrings`  |  检查值是字符串切片类型(`[]string`)
`ints/isInts`  |  检查值是`int` slice类型(only allow `[]int`)
`eq_field/eqField`  |  检查字段值是否等于另一个字段的值
`ne_field/neField`  |  检查字段值是否不等于另一个字段的值
`gte_field/gtField`  |  检查字段值是否大于另一个字段的值
`gt_field/gteField`  | 检查字段值是否大于或等于另一个字段的值
`lt_field/ltField`  |  检查字段值是否小于另一个字段的值
`lte_field/lteField`  |  检查字段值是否小于或等于另一个字段的值
`file/isFile`  |  验证是否是上传的文件
`image/isImage`  |  验证是否是上传的图片文件，支持后缀检查
`mime/mimeType/inMimeTypes`  |  验证是否是上传的文件，并且在指定的MIME类型中
`date/isDate` | 检查字段值是否为日期字符串。（只支持几种常用的格式） eg `2018-10-25`
`gt_date/gtDate/afterDate` | 检查输入值是否大于给定的日期字符串
`lt_date/ltDate/beforeDate` | 检查输入值是否小于给定的日期字符串
`gte_date/gteDate/afterOrEqualDate` | 检查输入值是否大于或等于给定的日期字符串
`lte_date/lteDate/beforeOrEqualDate` | 检查输入值是否小于或等于给定的日期字符串
`hasWhitespace` | 检查字符串值是否有空格
`ascii/ASCII/isASCII` | 检查值是ASCII字符串
`alpha/isAlpha` | 验证值是否仅包含字母字符
`alpha_num/alphaNum/isAlphaNum` | 验证是否仅包含字母、数字
`alpha_dash/alphaDash/isAlphaDash` | 验证是否仅包含字母、数字、破折号（ - ）以及下划线（ _ ）
`multi_byte/multiByte/isMultiByte` | 检查值是多字节字符串
`base64/isBase64` | 检查值是Base64字符串
`dns_name/dnsName/DNSName/isDNSName` | 检查值是DNS名称字符串
`data_uri/dataURI/isDataURI` | Check value is DataURI string.
`empty/isEmpty` | 检查值是否为空
`hex_color/hexColor/isHexColor` | 检查值是16进制的颜色字符串
`hexadecimal/isHexadecimal` | 检查值是十六进制字符串
`json/JSON/isJSON` | 检查值是JSON字符串。
`lat/latitude/isLatitude` | 检查值是纬度坐标
`lon/longitude/isLongitude` | 检查值是经度坐标
`mac/isMAC` | 检查值是MAC字符串
`num/number/isNumber` | 检查值是数字字符串. `>= 0`
`numeric/isNumeric` | 检查值是数值字符串(仅数字 `[0-9]+`, `>= 0`)。接受任意类型值，会先转为字符串再检查。
`strNum/stringNum/isStringNumber` | 检查值是数字字符串(仅数字 `[0-9]+`, `>= 0`)。仅接受字符串类型。
`cn_mobile/cnMobile/isCnMobile` | 检查值是中国11位手机号码字符串
`printableASCII/isPrintableASCII` | Check value is PrintableASCII string.
`rgbColor/RGBColor/isRGBColor` | 检查值是RGB颜色字符串
`full_url/fullUrl/isFullURL` | 检查值是完整的URL字符串(_必须以http,https开始的URL_).
`url/URL/isURL` | 检查值是URL字符串
`ip/IP/isIP`  |  检查值是IP（v4或v6）字符串
`ipv4/isIPv4`  |  检查值是IPv4字符串
`ipv6/isIPv6`  |  检查值是IPv6字符串
`cidr/CIDR/isCIDR` | 检查值是 CIDR 字符串
`CIDRv4/isCIDRv4` | 检查值是 CIDR v4 字符串
`CIDRv6/isCIDRv6` | 检查值是 CIDR v6 字符串
`uuid/isUUID` | 检查值是UUID字符串
`uuid3/isUUID3` | 检查值是UUID3字符串
`uuid4/isUUID4` | 检查值是UUID4字符串
`uuid5/isUUID5` | 检查值是UUID5字符串
`filePath/isFilePath` | 检查值是一个存在的文件路径
`pathExist/pathExists` | 检查值是一个存在的路径(文件或目录)。
`dirPath/isDirPath` | 检查值是一个存在的本地目录路径。
`unixPath/isUnixPath` | 检查值是Unix Path字符串
`winPath/isWinPath` | 检查值是Windows路径字符串
`isbn10/ISBN10/isISBN10` | 检查值是ISBN10字符串
`isbn13/ISBN13/isISBN13` | 检查值是ISBN13字符串

**提示**

- `intX` 包含: `int`, `int8`, `int16`, `int32`, `int64`
- `uintX` 包含: `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `floatX` 包含: `float32`, `float64`

<a id="built-in-filters"></a>
## 内置过滤器

> Filters powered by: [gookit/filter](https://github.com/gookit/filter)

过滤器/别名 | 描述信息 
-------------------|-------------------------------------------
`int/toInt`  | Convert value(string/intX/floatX) to `int` type `v.FilterRule("id", "int")`
`uint/toUint`  | Convert value(string/intX/floatX) to `uint` type `v.FilterRule("id", "uint")`
`int64/toInt64`  | Convert value(string/intX/floatX) to `int64` type `v.FilterRule("id", "int64")`
`float/toFloat`  | Convert value(string/intX/floatX) to `float` type
`bool/toBool`  | Convert string value to bool. (`true`: "1", "on", "yes", "true", `false`: "0", "off", "no", "false")
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

## 欢迎Star

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
