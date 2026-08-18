# Global Config Concurrency and Freeze Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复 #344 的全局 validator 并发竞争，并提供显式、不可逆的全部全局配置冻结能力。

**Architecture:** 使用一个包级 `sync.RWMutex` 保护全局配置修改边界。第一阶段只接入 validator 注册表；第二阶段加入冻结状态，并把 filter、message、custom type 纳入同一边界。公开 map 只返回快照，实例级配置保持独立。

**Tech Stack:** Go 1.21、标准库 `sync`/`os`/`os/exec`、`github.com/gookit/goutil/x/assert`、Docker `golang:1.25` race detector。

---

### Task 1: 修复 #344 全局 validator 并发竞争

**Files:**
- Create: `global_config_test.go`
- Modify: `register.go:3-10`
- Modify: `validators.go:157-183`
- Modify: `validation.go:413-493`
- Modify: `docs/superpowers/plans/2026-08-18-global-config-concurrency-and-freeze.md`

- [x] **Step 1: 写入可复现 issue 的并发测试**

在 `global_config_test.go` 定义带 `ConfigValidation` 的测试结构体。该方法故意调用包级 `AddValidator`；测试并发执行 `Struct(...).Validate()`，用 `atomic.Bool` 记录失败：

~~~go
package validate

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/gookit/goutil/x/assert"
)

const issue344Validator = "issue344NonBlank"

type issue344Payload struct {
	Name string `validate:"required|issue344NonBlank"`
}

func (issue344Payload) ConfigValidation(*Validation) {
	AddValidator(issue344Validator, func(val any) bool {
		s, ok := val.(string)
		return ok && s != ""
	})
}

func TestIssue344GlobalValidatorConcurrent(t *testing.T) {
	var failed atomic.Bool
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if !Struct(&issue344Payload{Name: "ok"}).Validate() {
					failed.Store(true)
				}
			}
		}()
	}
	wg.Wait()
	assert.False(t, failed.Load())
}
~~~

- [x] **Step 2: 用 Go 1.25 race detector 证明测试失败**

Run:

~~~powershell
docker run --rm --mount "type=bind,source=$PWD,target=/src,readonly" -w /src golang:1.25 sh -c 'go test -race -run ^TestIssue344GlobalValidatorConcurrent$ .'
~~~

Expected: FAIL，并出现 `WARNING: DATA RACE` 或 `fatal error: concurrent map writes`，写端指向包级 `AddValidator`。

- [x] **Step 3: 加入最小 validator 全局锁**

在 `register.go` 导入 `sync` 并声明：

~~~go
var globalConfigMu sync.RWMutex
~~~

在 `validators.go` 增加仅供持有写锁时调用的 helper；单个和批量注册各自在一次写锁内完成：

~~~go
func addGlobalValidator(name string, checkFunc any) {
	fv := checkValidatorFunc(name, checkFunc)
	validators[name] = validatorTypeCustom
	validatorMetas[name] = newFuncMeta(name, false, fv)
}

func AddValidators(m map[string]any) {
	globalConfigMu.Lock()
	defer globalConfigMu.Unlock()
	for name, checkFunc := range m {
		addGlobalValidator(name, checkFunc)
	}
}

func AddValidator(name string, checkFunc any) {
	globalConfigMu.Lock()
	defer globalConfigMu.Unlock()
	addGlobalValidator(name, checkFunc)
}

func Validators() map[string]int8 {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	cp := make(map[string]int8, len(validators))
	for name, typ := range validators {
		cp[name] = typ
	}
	return cp
}
~~~

在 `validation.go` 只围绕全局 map 访问加读锁：

~~~go
globalConfigMu.RLock()
fm, ok := validatorMetas[name]
globalConfigMu.RUnlock()
if ok {
	return fm
}
~~~

`HasValidator` 使用相同读锁；`Validation.Validators(true)` 在读锁内遍历全局 `validators`，释放后再合并实例 map。

- [x] **Step 4: 格式化并验证定向测试转绿**

Run:

~~~powershell
gofmt -w register.go validators.go validation.go global_config_test.go
go test -run ^TestIssue344GlobalValidatorConcurrent$ .
docker run --rm --mount "type=bind,source=$PWD,target=/src,readonly" -w /src golang:1.25 sh -c 'go test -race -run ^TestIssue344GlobalValidatorConcurrent$ .'
~~~

Expected: 全部 PASS，Docker 输出没有 `DATA RACE`。

- [x] **Step 5: 执行阶段一全量验证**

Run: `go test ./...`，然后运行 `go vet ./...`。

Expected: 全部退出码为 0。

- [x] **Step 6: 更新本任务 checkbox 并提交阶段一**

Run:

~~~powershell
git add -- register.go validators.go validation.go global_config_test.go docs/superpowers/plans/2026-08-18-global-config-concurrency-and-freeze.md
git commit -m "fix(validate): protect global validators from concurrent access (fix #344)"
~~~

Expected: 一个只包含阶段一代码、测试和进度更新的提交。

### Task 2: 新增显式 `FreezeGlobal()`

**Files:**
- Modify: `register.go`
- Modify: `validators.go`
- Modify: `filtering.go:12-66`
- Modify: `messages.go:271-294,342-350`
- Modify: `register_type.go:30-52`
- Modify: `global_config_test.go`
- Modify: `docs/superpowers/plans/2026-08-18-global-config-concurrency-and-freeze.md`

- [ ] **Step 1: 写入子进程冻结测试**

扩展 `global_config_test.go` 的导入。父测试启动当前测试二进制的单一 helper；子进程验证幂等冻结、全部全局修改 API panic，以及实例级配置仍可用：

~~~go
func TestFreezeGlobal(t *testing.T) {
	if os.Getenv("GO_WANT_FREEZE_GLOBAL_HELPER") == "1" {
		FreezeGlobal()
		FreezeGlobal()

		assert.Panics(t, func() { AddValidator("frozenValidator", func(any) bool { return true }) })
		assert.Panics(t, func() { AddValidators(M{"frozenValidators": func(any) bool { return true }}) })
		assert.Panics(t, func() { AddFilter("frozenFilter", func(val any) any { return val }) })
		assert.Panics(t, func() { AddFilters(M{"frozenFilters": func(val any) any { return val }}) })
		assert.Panics(t, func() { AddGlobalMessages(MS{"frozen": "frozen"}) })
		assert.Panics(t, func() { AddBuiltinMessages(MS{"frozen": "frozen"}) })
		assert.Panics(t, func() { SetBuiltinMessages(MS{"frozen": "frozen"}) })
		assert.Panics(t, func() { AddCustomType(func(reflect.Value) any { return nil }, struct{}{}) })
		assert.Panics(t, ResetCustomTypes)

		v := Map(M{"name": "ok"})
		assert.NotPanics(t, func() { v.AddValidator("localValidator", func(any) bool { return true }) })
		assert.NotPanics(t, func() { v.AddFilter("localFilter", func(val any) any { return val }) })
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestFreezeGlobal$")
	cmd.Env = append(os.Environ(), "GO_WANT_FREEZE_GLOBAL_HELPER=1")
	out, err := cmd.CombinedOutput()
	assert.NoErr(t, err, string(out))
}
~~~

- [ ] **Step 2: 运行测试证明 API 尚不存在**

Run: `go test -run ^TestFreezeGlobal$ .`

Expected: 编译失败并包含 `undefined: FreezeGlobal`。

- [ ] **Step 3: 实现冻结状态和统一修改边界**

在 `register.go` 增加：

~~~go
var globalFrozen bool

// FreezeGlobal prevents further changes to package-level configuration.
func FreezeGlobal() {
	globalConfigMu.Lock()
	globalFrozen = true
	globalConfigMu.Unlock()
}

func panicIfGlobalFrozen() {
	if globalFrozen {
		panic("validate: global configuration is frozen")
	}
}
~~~

所有全局修改 API 在 `globalConfigMu` 写锁内先调用 `panicIfGlobalFrozen()`。批量 API在一次写锁内完成整个批次，避免与冻结交错产生部分写入。

`filtering.go` 为包级 `AddFilter(s)` 增加私有 `addGlobalFilter` helper；实例方法保持原状。`FilterFuncValue` 只在读取全局 `filterValues` 时获取读锁。

`messages.go` 的公开 map 出口改为快照：

~~~go
func BuiltinMessages() map[string]string { return CopyGlobalMessages() }

func CopyGlobalMessages() map[string]string {
	globalConfigMu.RLock()
	defer globalConfigMu.RUnlock()
	cp := make(map[string]string, len(builtinMessages))
	for name, msg := range builtinMessages {
		cp[name] = msg
	}
	return cp
}
~~~

`AddGlobalMessages`、`SetBuiltinMessages` 获取写锁并检查冻结；`SetBuiltinMessages` 手工复制输入 map 后保存。`Translator.lookupMessage` 只在读取全局 `builtinMessages` 时获取读锁。`AddBuiltinMessages` 继续复用 `AddGlobalMessages`。

`register_type.go` 的 `AddCustomType`、`ResetCustomTypes` 获取同一写锁并检查冻结；底层 `sync.Map` 和 `atomic.Bool` 保持不变。

- [ ] **Step 4: 格式化并验证冻结测试转绿**

Run:

~~~powershell
gofmt -w register.go validators.go filtering.go messages.go register_type.go global_config_test.go
go test -run ^TestFreezeGlobal$ .
docker run --rm --mount "type=bind,source=$PWD,target=/src,readonly" -w /src golang:1.25 sh -c 'go test -race -run "^(TestFreezeGlobal|TestIssue344GlobalValidatorConcurrent)$" .'
~~~

Expected: 全部 PASS，无 data race。

- [ ] **Step 5: 执行阶段二全量验证**

Run: `go test ./...`，然后运行 `go vet ./...`。

Expected: 全部退出码为 0。

- [ ] **Step 6: 更新全部 checkbox 并提交阶段二**

Run:

~~~powershell
git add -- register.go validators.go filtering.go messages.go register_type.go global_config_test.go docs/superpowers/plans/2026-08-18-global-config-concurrency-and-freeze.md
git commit -m "feat(validate): add explicit global configuration freeze"
~~~

Expected: 第二个实现提交只包含冻结能力、相关并发保护、测试和最终进度更新。
