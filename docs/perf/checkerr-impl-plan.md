# `CheckErr` 实施计划（opt-in 快速过/败入口，目标 3 allocs）

> **类型**：实施计划（**已完成 ✅** —— P1 机制 + P2 测试/基准/文档均已落地）。
> **日期**：2026-06-07
> **前置**：`v2.x-perf-optimization-plan.md`（S1–S3.1 已落地，`Check` 池化 = 6 allocs）、`t3-deboxing-design.md` §6.1（CheckErr spike 实测：A=6 / B=8 / **C=3**）。
> **范围**：仅 **struct 数据源**。新增公开函数 `CheckErr`，**附加、非破坏**。

---

## 0. 目标与结论先行

为"只要过/败、不取清洗数据/不绑定"的高频校验（如中间件快速 reject）提供一个 opt-in 入口：

```go
func CheckErr(structPtr any, scene ...string) error // 通过 nil；失败返回 Errors.OneError()
```

- 内部走包级 `defaultFactory` 池（同 `Check`），**跳过 safeData/filteredData 收集 + 不建 ValidResult 壳**。
- 用 **1 槽"提交值"缓存**补回 safeData 隐含的"同字段装箱去重"，否则会退化为负优化（spike B：6→8）。
- 预期 **3 allocs / op**（spike C 实测：6→3，B/op −90%，ns −16%），行为相对 `Check` **逐字节一致**（golden 守护）。

---

## 1. 已验证数据（spike，代码已回退）

bench `gookitFlat`（Name 3 规则 / Email 2 / Age 3）：

| 变体 | allocs | ns/op | B/op |
|---|---:|---:|---:|
| A `Check`（基线） | 6 | 1389 | 405 |
| B `CheckErr` 无去重 | **8 ⚠️** | 1428 | 105 |
| C `CheckErr` + 1槽去重 | **3** | 1175 | 40 |

B 负优化的根因：跳过 safeData 丢了它隐含的装箱去重，8 条规则各自 `TryGet→fv.Interface()` 重新装箱。C 补回去重即兑现"省 3 个"。

---

## 2. API 设计

| 项 | 决策 |
|---|---|
| 签名 | `func CheckErr(structPtr any, scene ...string) error` |
| 返回 | 通过 `nil`；失败 `v.Errors.OneError()`（与 `ValidateErr` 语义一致；失败路径 1 alloc 来自 `errorx.Raw`，可接受） |
| 数据源 | **仅 struct**（map/builder 维持现状 `New/Map + ValidateErr`）。非 struct 传入走 `FromStruct` 既有错误路径（返回 `ErrInvalidData` 包装的 error） |
| 破坏性 | **无**。纯新增公开符号；`Check`/`ValidateR`/`Validation`/`ValidResult` 全不动 |
| 内部状态 | 给 `Validation` 加**未导出**字段（不算 API） |
| 文档定位 | "只要过/败、不要清洗数据/绑定"的快速入口；要数据/绑定仍用 `Check`/`ValidateR` |

> 命名备选：`CheckErr`（推荐，对齐 `ValidateErr`）。不取 `CheckE`（与 `ValidateE` 返回 `Errors` 的语义不同，避免混淆）。

---

## 3. 内部机制设计

### 3.1 新增未导出字段（`validation.go` Validation 结构体）

```go
skipCollect bool   // CheckErr 模式：跳过 safeData/filteredData 收集
scKey       string // 1 槽去重：上一次"提交"的字段名（""=空槽）
scVal       any    // 1 槽去重：上一次"提交"的值(已解析/过滤)
```

### 3.2 `CheckErr` 入口（`result.go` 或新文件）

```go
func CheckErr(structPtr any, scene ...string) error {
    v := defaultFactory.Struct(structPtr, scene...) // 池化实例 + UpdateSource=true
    v.skipCollect = true                            // 必须在 Validate 前置位
    v.Validate()
    err := v.Errors.OneError()
    v.Release()                                      // 归还池(resetForReuse 清 skipCollect/槽)
    return err
}
```

### 3.3 收集门控（`validating.go applyField`）

把 3 个"提交"写入点（isDefault 存 :201-202、filter 存 :232-233、validate 通过存 :246-247）改为：skipCollect 模式写 1 槽，否则维持原 safeData/filteredData 写入。示意（validate 通过点）：

```go
if v.skipCollect {
    v.scKey, v.scVal = field, val      // 提交值进 1 槽
} else {
    v.ensureSafeData()
    v.safeData[field] = val
}
```

### 3.4 去重读取（`validation.go tryGet`）

```go
func (v *Validation) tryGet(key string) (val any, exist, zero bool) {
    if v.data == nil { return }
    if v.skipCollect {
        if v.scKey == key {            // 同字段连续读 → 复用提交值,免重新装箱
            return v.scVal, true, false // zero=false 与 safeData 命中语义一致
        }
        return v.data.TryGet(key)      // 其它字段 → 落源(源已写回解析值,见 §4)
    }
    if val1, ok := v.filteredData[key]; ok { return val1, true, false }
    if val2, ok := v.safeData[key];     ok { return val2, true, false }
    return v.data.TryGet(key)
}
```

### 3.5 复用清理（`validation.go resetForReuse`）

末尾加：`v.skipCollect = false; v.scKey = ""; v.scVal = nil`。**关键**：否则同一池实例被 `CheckErr` 用后再被 `Check` 取用会残留 `skipCollect=true` → `Check` 收不到 safeData。

---

## 4. 正确性论证（为何跳过 safeData 仍与 `Check` 等价）

`Check` 的 safeData 在校验期承担两件事：**(a) 同字段多规则的装箱去重**、**(b) 给跨字段校验器（requiredWith/eqField…）提供"已解析(默认/过滤后)值"**。逐一对应：

1. **(b) 跨字段读 → 靠源写回，不依赖 safeData。** struct 源 `UpdateSource=true`（`createInto` 设，`data_source.go:304`），`updateValue → StructData.Set` 把**默认值/过滤值写回源 struct**（`validation.go updateValue`）。S1 后 `TryGet` 每次 `FieldByIndex` 重读源 → 跨字段 `tryGet` 落源即读到**已解析值**，与 `Check` 读 safeData 一致。→ CheckErr 跳过 safeData **不影响跨字段校验结果**。
2. **(a) 同字段连续读 → 1 槽替代。** struct tag 驱动下 `parseRulesFromTag` 按字段顺序逐个 `StringRule`，**同字段多规则在 `v.rules` 中连续**，故 `tryGet(同 key)` 连续命中 1 槽，把每字段装箱压回 1 次（= safeData 的去重效果）。
3. **zero 语义一致。** `Check` 命中 safeData/filteredData 时返回 `zero=false`；1 槽命中也返回 `zero=false`。两者命中的都是**提交后(已过默认/过滤)值**，故 `GetWithDefault`/`skipEmpty` 分支行为一致。
4. **退化安全。** 若同字段规则不连续（手工 AddRule 交叉等），1 槽 miss → 落源 → 读到源里的已解析值（写回保证）→ **仍正确**，只是该字段多装箱一次。即 1 槽是纯性能优化，**不影响正确性下限**。

> ⚠️ 边界澄清：写回只对 struct 源生效（`updateValue` 对 form/map 返回原值不写回）——这正是把 `CheckErr` 限定 **struct-only** 的原因。map/builder 不走此入口。

---

## 5. 改造点逐一（按文件 + 位置）

| # | 文件 | 位置 | 改动 |
|---|---|---|---|
| C-1 | `validation.go` | Validation struct | 加 `skipCollect bool` / `scKey string` / `scVal any` |
| C-2 | `validation.go` | `tryGet`(:565) | skipCollect 分支：1 槽命中返回；否则落源（跳过 safeData/filteredData 读） |
| C-3 | `validation.go` | `resetForReuse`(:221) | 末尾清 `skipCollect`/`scKey`/`scVal` |
| C-4 | `validating.go` | `applyField` 3 处提交点(:201-202 / :232-233 / :246-247) | skipCollect 模式写 1 槽，否则原逻辑 |
| C-5 | `result.go`(或新 `checkerr.go`) | — | 新增 `CheckErr` |
| C-6 | `result_test.go` | — | 正确性/复用/并发/zero/filter/默认值/失败路径专项（§6 测试矩阵） |
| C-7 | `_examples/bench-vs-goplayground/bench_test.go` | — | `BenchmarkGookitCheckErrValid` / `...Invalid` |
| C-8 | `README.md` / `README.zh-CN.md` / 本计划 + 主计划文档 | — | 文档化 opt-in 入口 + 基准数字 |

---

## 6. 测试矩阵（C-6，golden 之外的专项）

每条都对照"等价 `Check` 结果"断言（过/败 + 失败字段集合一致）：

| 用例 | 验证点 |
|---|---|
| valid struct | `CheckErr`==nil；`Check` 同样 IsOK |
| invalid struct（多字段失败） | `CheckErr`!=nil；OneError 属于失败集合 |
| 多规则同字段（Name 3 规则） | 与 Check 结果一致；alloc 不因去重而错 |
| **zero 值 + 默认值**（field 有 default，源值为零） | 默认值生效、`isDefault` 路径与 Check 一致 |
| **SkipOnEmpty + 空值** 多规则字段 | skip 行为与 Check 一致 |
| **filter + 多规则同字段** | 过滤值写回源、后续规则见过滤值，与 Check 一致 |
| **跨字段校验器**（requiredWith / eqField，引用被 filter/default 的字段） | 结果与 Check 一致（验证 §4 写回论证） |
| 池复用：`CheckErr` 后接 `Check`（同/异类型） | `Check` 仍正常收 safeData（验证 resetForReuse 清 skipCollect） |
| 池复用：`CheckErr` 后接 `CheckErr`（异类型） | 1 槽不串味 |
| 并发：goroutine 混跑 `Check`/`CheckErr`（-race） | 无数据竞争、结果正确 |
| 非 struct 传入（map/nil/非结构体） | 返回 error（`ErrInvalidData` 路径），不 panic |

---

## 7. 分阶段实施（每步独立 commit + golden×3 + race）

- **P1（机制）✅ 已完成**：C-1～C-5。新增 `CheckErr` + skipCollect 门控 + 1 槽去重 + resetForReuse 清理。
  验收：`go build` + `go vet` + `go test -run TestRuleCompat -count=3 .`（golden 逐字节）+ `go test -count=1 .` + `-race` 全过。基准 `CheckErr` valid = 3 allocs ✅。
- **P2（测试与文档）✅ 已完成**：C-6 测试矩阵 + C-7 基准 + C-8 文档。
  验收：新用例全过 + `-race` + 基准数字入文档（见 §10 实施结果）。

> 体量评估：业务代码 ~5 处小改 + 1 新函数（远低于 100 行业务代码、3 文件量级以"逻辑点"计虽达 4 文件但均为小改），属附加特性。按规范分两子阶段提交。

---

## 8. 风险与回滚

| 风险 | 缓解 |
|---|---|
| 跳过 safeData 致跨字段校验分叉 | §4 写回论证 + 测试矩阵"跨字段"用例；golden×3 兜底 |
| resetForReuse 漏清 skipCollect → 污染 `Check` | C-3 显式清理 + "CheckErr 后接 Check"复用用例 |
| 1 槽 zero 语义偏差（默认值/skipEmpty） | 命中返回 `zero=false` 镜像 safeData；zero/默认值/skipEmpty 专项用例 |
| 同字段规则非连续致去重失效 | 仅性能退化（不影响正确性，§4-4）；不阻断 |
| 失败路径 `OneError` 的 alloc | 仅失败时 1 alloc（`errorx.Raw`），文档说明；可后续评估 `ErrOrNil` 复用 |

回滚：所有改动可单 commit revert；`CheckErr` 为附加 API，移除不影响其它路径。

---

## 9. 验收标准（缺一不可）

1. `go build ./...` + `go vet .` 干净。
2. `go test -run TestRuleCompat -count=3 .` 逐字节 golden 全过（证明 `CheckErr` 与 `Check` 行为一致的间接保证 + 既有路径不回归）。
3. `go test -count=1 .` + `go test -race -count=1 .` 全过（含 §6 新用例）。
4. 基准 `BenchmarkGookitCheckErrValid` = **3 allocs/op**（对照 `Check` 6）。
5. 文档（README 中英 + 本计划 + 主计划）同步，基准数字入文档。

---

## 10. 实施结果（P1 + P2 实测）

> 环境：go1.25.10，Intel i7-14700KF，`flat` 数据集（Name 3 规则 / Email 2 / Age 3）。
> 基准命令：`cd _examples/bench-vs-goplayground && GOWORK=off go test -bench 'CheckErr' -benchmem -run '^$' -benchtime=2s .`

### 10.1 基准数字

| 入口 | 路径 | allocs/op | B/op | ns/op |
|---|---|---:|---:|---:|
| `Check`（基线） | valid | 6 | 405 | ~1389 |
| **`CheckErr`** | **valid** | **3** ✅ | **40** | **~1133** |
| `Struct().Validate()` | invalid | 32 | 3106 | ~2035 |
| **`CheckErr`** | **invalid** | **21** | **1771** | **~1811** |

- **valid 路径**：兑现 spike C 目标 = **3 allocs/op**（对照 `Check` 6，B/op −90%、ns −18%）。
- **invalid 路径**：21 allocs（对照 `Struct().Validate()` 32），失败时的分配主要来自错误消息构建（3 个失败字段各自走消息模板/格式化），属固有成本；skipCollect 仍显著优于普通入口。

### 10.2 失败路径优化（本次落地）

`validating.go Validate()` 末尾：

```go
v.hasValidated = true
if v.hasError && !v.skipCollect { // clear safe data on error (skip in CheckErr fast path).
    v.safeData = make(map[string]any)
}
```

skipCollect 模式失败时 `safeData` 维持 nil（CheckErr 不读它，`tryGet` skip 分支与 `resetForReuse` 的 `clear(nil)` 均安全），省去这次无谓 `make` —— 让 CheckErr 失败路径也少 1 alloc。非 skipCollect 路径完全不变（golden×3 守护）。

### 10.3 测试矩阵（result_test.go，全过）

核心策略 **differential testing**：对一组多样结构体（合法/非法/多规则/含 filter tag/含跨字段规则各若干）断言 `(CheckErr(s)==nil) == Check(s).IsOK()`。覆盖：

| 用例 | 结果 |
|---|---|
| `TestCheckErr_vsCheck_differential`（20 例工厂） | ✅ |
| 多规则同字段（required/min_len/max_len 各失败） | ✅ |
| filter + 多规则同字段（trim\|lower 后过/败一致） | ✅ |
| 跨字段 `eq_field`/`required_with`/`gt_field` 引用带 filter 字段 | ✅ |
| 默认值 `default:` tag（零值用默认、显式非零覆盖默认） | ✅ |
| skipEmpty 语义（空可选字段跳过、非空校验） | ✅ |
| 池复用 CheckErr→CheckErr 异类型交替 | ✅ |
| 并发混跑 Check/CheckErr（-race） | ✅ |
| 非 struct 入参（nil/map/int/string → error，不 panic） | ✅ |

> **关键发现（非 bug，方法论守门）**：differential 必须**每次新建结构体**。struct 源 `UpdateSource=true`，默认值/过滤值写回源；若复用同一指针先 `CheckErr` 再 `Check`，第二次校验看到的是已被改写的源（如默认值字段已非零 → 改走 filter 路径），导致两次结果"看似不一致"。这是 gookit 既有源写回语义（两次连续 `Check` 同指针亦然），与 CheckErr 无关。测试用例已用工厂函数 `func() any` 保证每次产新实例。

### 10.4 自检结果

`go build ./...` + `go vet .` 干净；`go test -run TestRuleCompat -count=3 .`（golden 逐字节）、`go test -count=1 .`、`go test -race -count=1 .` 全过。
