# 全局配置并发保护与显式冻结设计

## 目标

分两个独立阶段完成：

1. 修复 #344，保证全局 validator 注册与读取并发安全。
2. 新增 `FreezeGlobal()`，由应用在初始化完成后显式冻结全部全局配置。

不自动推断应用启动结束时间，不改变实例级配置能力。

## 阶段一：修复 #344

使用一个包级 `sync.RWMutex` 同时保护 `validators` 和 `validatorMetas`：

- `AddValidator` 在一次写锁内更新两张表。
- `AddValidators` 复用相同的受保护注册逻辑。
- `validatorMeta`、`HasValidator` 和 `Validation.Validators(true)` 在读取全局表时使用读锁。
- 包级 `Validators()` 在读锁内返回快照，不再暴露内部 map。

内置 validator 仍在包初始化期间注册，不改变现有初始化顺序。实例级
`Validation.AddValidator` 不访问全局表，不需要该锁。

新增最小并发回归测试，同时执行全局注册和 validator 查询/校验；测试必须在
`go test -race` 下无 data race。提交信息包含 `fix #344`。

## 阶段二：显式冻结全部全局配置

新增：

```go
func FreezeGlobal()
```

语义：

- 调用由应用显式完成，不在首次校验时自动触发。
- 可重复调用，重复冻结不报错。
- `FreezeGlobal()` 返回后，任何全局配置修改 API 都立即 panic。
- 不提供解冻 API，也不增加冻结状态查询 API。
- 实例级 `v.AddValidator`、`v.AddFilter`、translator 实例配置继续可用。

冻结范围包含：

- validator：`AddValidator`、`AddValidators`
- filter：`AddFilter`、`AddFilters`
- message：`AddGlobalMessages`、`AddBuiltinMessages`、`SetBuiltinMessages`
- custom type：`AddCustomType`、`ResetCustomTypes`

冻结状态和上述修改操作共用同一个同步边界，保证冻结与注册并发发生时只有明确
的先后顺序：先完成的注册保留；`FreezeGlobal()` 返回后不能再写入。

为避免从公开 map 绕过冻结：

- `Validators()` 返回快照。
- `BuiltinMessages()` 返回快照。
- `SetBuiltinMessages()` 保存输入 map 的副本。

统一使用清晰的冻结 panic 信息。实例级 API 不检查全局冻结状态。

## 验证

阶段一：

- 定向测试覆盖并发注册、读取和实际校验。
- Docker `golang:1.25` 执行定向 `go test -race`。
- 执行 `go test ./...`、`go vet ./...`。

阶段二：

- 验证重复冻结。
- 验证四类全局修改 API 冻结后全部 panic。
- 验证实例级 validator/filter 仍可添加。
- 冻结测试在子进程中运行，避免不可逆全局状态污染其他测试。
- 执行 `go test ./...`、`go vet ./...`，并用 Docker Go 1.25 执行相关 race 测试。

每个阶段分别提交，只包含该阶段直接相关的代码和测试。
