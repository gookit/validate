# 升级指南 / Upgrade Guide: v1.x → v2.0

> 本文面向已经在使用 `github.com/gookit/validate` v1.x 的用户。
> This guide targets users upgrading from `github.com/gookit/validate` v1.x.

v2.0 在设计上**刻意保持最小破坏**：核心 API、验证器名称、tag 语义、`DataFace`
接口都保持不变。绝大多数项目只需要改 import 路径与执行 `go get` 即可完成升级；
仅当你直接调用了 `Between` / `ValueLen` 函数时才需要额外改动。

v2.0 keeps breaking changes intentionally minimal: core API, validator names, tag
semantics and the `DataFace` interface are all unchanged. Most projects only need
to bump the import path. Extra work is required **only** if your code calls
`Between` or `ValueLen` directly.

---

## 一分钟升级 / One-minute upgrade

```bash
# 1. 拉取 v2 模块 / pull the v2 module
go get github.com/gookit/validate/v2

# 2. 全局替换 import 路径 / rewrite import paths
#    github.com/gookit/validate      -> github.com/gookit/validate/v2
#    github.com/gookit/validate/...  -> github.com/gookit/validate/v2/...

# 3. 整理依赖 / tidy
go mod tidy
```

替换示例 / import example:

```go
// v1.x
import "github.com/gookit/validate"
import "github.com/gookit/validate/locales/zhcn"

// v2.0
import "github.com/gookit/validate/v2"
import "github.com/gookit/validate/v2/locales/zhcn"
```

> 包名仍是 `validate`，代码中 `validate.Struct(...)` 等调用无需改动，只改 import 行。
> The package name is still `validate`; only the import path changes, not call sites.

---

## 破坏性变更 / Breaking Changes（共 5 项 / 5 total）

### 1. 模块路径加 `/v2`（Module path）

遵循 Go Modules 的语义化版本规范，v2 模块路径带 `/v2` 后缀。

Per Go Modules semantic import versioning, the v2 module path carries a `/v2` suffix.

```diff
- go get github.com/gookit/validate
- import "github.com/gookit/validate"
+ go get github.com/gookit/validate/v2
+ import "github.com/gookit/validate/v2"
```

**Why**：Go Modules 要求 v2+ 主版本在模块路径中显式带 `/vN`，否则无法与 v1 共存、
也无法被正确解析。

### 2. 最低 Go 版本 1.19 → 1.21（Minimum Go version）

`go.mod` 的最低要求从 `go 1.19` 提升到 `go 1.21`。请确保你的构建环境 Go ≥ 1.21。

```diff
- go 1.19
+ go 1.21
```

**Why**：使用了 1.21 起稳定的标准库与语言能力，并与 goutil 等依赖的基线对齐。

### 3. `Between` 签名改为全 `any`（Between signature）

`Between` 由「整数边界」改为与 `Gt` / `Lt` 一致的「任意可比较值」，统一走
`valueCompare`，支持 int / uint / float / string。

```diff
- func Between(val any, min, max int64) bool
+ func Between(val, min, max any) bool
```

直接调用方一般无需改动（整型边界仍可用）；但**小数边界的语义发生变化**：

```go
// v1.x: min/max 为 int64，2.9 会被整数截断为 2，落入 [1,2] -> true
validate.Between(2.9, 1, 2) // => true

// v2.0: 不再截断，2.9 > 2 -> false（更符合直觉）
validate.Between(2.9, 1, 2) // => false
```

通过 tag / StringRule 使用 `between:1,2` 的方式不受影响。

**Why**：v1 把边界强制为 `int64`，会把浮点边界与浮点值悄悄截断为整数，导致
`2.9` 落入 `[1,2]` 这种反直觉结果。v2 与 `Gt`/`Lt` 统一为 `any`，按实际数值比较。

### 4. 删除 deprecated 的 `ValueLen`（Removed `ValueLen`）

已废弃的 `ValueLen(v)` 在 v2.0 移除，请改用 goutil 的 `reflects.Len`。

```diff
- import "github.com/gookit/validate"
- n := validate.ValueLen(reflect.ValueOf(v))
+ import "github.com/gookit/goutil/reflects"
+ n := reflects.Len(reflect.ValueOf(v))
```

**Why**：该函数早已标记 deprecated，其实现本就转发到 goutil；直接用 goutil 的
`reflects.Len` 去掉了重复封装。`reflects.Len` 接收 `reflect.Value`，返回 `int`。

> **未变更提示**：v2.0 设计期曾计划调整 `DataFace` 接口，profile 后否决——
> `DataFace` 接口**保持不变**，自定义 `DataFace` 实现者**不受影响**。
> Note: the `DataFace` interface is **unchanged** in v2.0; custom implementers are unaffected.

### 5. 子结构体级联默认需父字段带 `validate` tag（Sub-struct cascade opt-in）

v1.x 对子结构体字段（`struct` / `*struct` / slice-of-struct / map-of-struct）**无条件递归**
收集其内部字段规则。v2.0 改为 Java `@Valid` 风格的「按需下探」：仅当父字段带有
`validate` tag（值可为空）时才级联验证其**具名**子结构；具名字段完全没有 `validate`
tag 时**不再**下探。该行为由 `CheckSubOnParentMarked` 控制，v2 默认 **true**。
**匿名嵌入结构体**（`type Bar struct { Foo }`，字段被提升、属父结构体一部分）**豁免**——
始终级联，无需 tag；只有**具名**嵌套字段需要标记。

In v1.x a sub-struct field was **always** descended into to collect its inner
rules. v2.0 makes this opt-in (Java `@Valid` style): cascade happens **only** when
the parent field carries a `validate` tag. An **empty** tag (`validate:""`) is
enough to mark it; a **named** field with **no** `validate` tag is no longer
descended into. **Anonymous embedded structs are exempt** (they are part of the
parent and always cascade). Controlled by `CheckSubOnParentMarked`, default **true** in v2.

```go
type Address struct {
	City string `validate:"required"`
	Zip  string `validate:"required|minLen:3"`
}

type User struct {
	Name string `validate:"required"`

	// v1.x: Address 内部的 City/Zip 规则会被自动收集并校验
	// v2.0: Addr 无 validate tag -> 不再下探，City/Zip 不校验
	Addr Address
}
```

**迁移方法 / Migration**——二选一：

1. **给需要级联的无 tag 嵌套字段补一个 `validate` tag**（推荐，按字段精确控制）。
   空 tag `validate:""` 即可恢复级联，无需任何规则：

   ```diff
   type User struct {
   	Name string `validate:"required"`
   -	Addr Address
   +	Addr Address `validate:""`     // 空 tag 即可重新开启下探
   	// 或者给父字段本身加规则，如 `validate:"required"`，同样会下探
   }
   ```

   > 注意：匿名嵌入字段（`User { Address }`）同样适用——需在嵌入行加 `validate:""` 才会下探。

2. **全局恢复 v1「永远级联」行为**（一处改动覆盖全部类型，适合不想逐字段标注的大型工程）：

   ```go
   validate.Config(func(o *validate.GlobalOption) {
   	o.CheckSubOnParentMarked = false
   })
   ```

**Why**：v1 的无条件递归会在你只想校验顶层字段时，意外地把深层子结构体的规则也
一并收集执行，且无法关闭。v2 改为显式标记后下探，行为更可预期，同时保留全局逃生舱。

---

## 新增能力 / New Features

### `AddCustomType`：为自定义类型注册值提取器

为 `sql.NullString`、money 包装类型等「带壳」的自定义类型注册一个提取器，把它的
底层值取出来后，照常走现有的校验路径（`required` / 数值比较 / 长度 / 字符串规则）。
对标 go-playground/validator 的 `RegisterCustomTypeFunc`。

```go
package main

import (
	"database/sql"
	"reflect"

	"github.com/gookit/validate/v2"
)

func main() {
	// 注册：从 sql.NullString 提取底层字符串；无效则返回 nil（视为空）。
	validate.AddCustomType(func(field reflect.Value) any {
		ns := field.Interface().(sql.NullString)
		if !ns.Valid {
			return nil // 返回 nil 表示"空/未设置"，required 将判定失败
		}
		return ns.String
	}, sql.NullString{})

	type Form struct {
		Name sql.NullString `validate:"required|minLen:3"`
	}

	v := validate.Struct(&Form{Name: sql.NullString{Valid: true, String: "inhere"}})
	_ = v.Validate() // true：提取出 "inhere" 后按 required|minLen:3 校验

	// 测试 / 清理可用 ResetCustomTypes() 复位注册表。
	// validate.ResetCustomTypes()
}
```

说明：
- 签名 `AddCustomType(fn CustomTypeFunc, types ...any)`，`CustomTypeFunc func(field reflect.Value) any`。
- 提取器返回 `nil` 表示该值为「空/未设置」，`required` 会失败。
- 按样例的精确 `reflect.Type` 匹配，**不自动解指针**：传 `sql.NullString{}` 只匹配值类型；
  若要同时匹配 `*sql.NullString`，需再传入 `&sql.NullString{}` 样例。
- `ResetCustomTypes()` 清空全部注册（测试 / 清理用）。
- 未注册任何自定义类型时，校验热路径零额外开销（原子门控短路）。

### `NewFactory`：opt-in 池化工厂（重用场景降分配）

`Struct` / `Map` / `New` 的默认行为与生命周期**完全不变**。如果你在循环里高频校验
**同类型**数据，可以显式用 `Factory` 复用 `*Validation` 实例，摊销构造成本——
复用场景的 allocs 约减半。

```go
package main

import "github.com/gookit/validate/v2"

type User struct {
	Name  string `validate:"required|minLen:3"`
	Email string `validate:"required|email"`
}

func main() {
	users := []User{ /* ... 大量同类型数据 ... */ }

	f := validate.NewFactory()
	for i := range users {
		v := f.Struct(&users[i]) // 从池中取复用实例 + 复用类型元数据
		v.Validate()
		// ... 使用 v.Errors / v.SafeData() ...
		v.Release() // 必须调用：Reset 后归还到池
	}
}
```

注意事项：
- 从 `Factory` 取得的 `*Validation` 行为与 `validate.Struct` / `validate.Map` 完全一致。
- **必须**在用完后调用 `v.Release()` 归还实例；`Release()` 之后**不要**再使用该实例。
- `Release()` 对非工厂来源的实例（`v.pool == nil`）是 no-op，任何时候调用都安全。
- 这是 opt-in 优化，不影响既有代码——只有显式使用 `Factory` 的调用方才会走池化路径。

---

## 升级后自检 / Post-upgrade checklist

```bash
go build ./...
go vet ./...
go test ./...
```

若以上全部通过，且你没有直接调用 `Between(float, ...)` 或 `ValueLen`、也没有依赖
子结构体的「无 tag 自动级联」（见破坏性变更第 5 项），则升级完成。若你的嵌套字段
校验在升级后「不生效」了，多半是命中了第 5 项：给该嵌套字段补 `validate:""`，或全局
设 `CheckSubOnParentMarked = false`。
