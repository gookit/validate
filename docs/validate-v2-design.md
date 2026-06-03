# gookit/validate V2 重构设计方案

> **版本**: v2.0-design (rev.4)
> **日期**: 2026-06-03
> **状态**: v1.6.0 已落地（见 §0.1 实现状态对账）；v2.0 破坏性部分仍待启动
> **替代**: `design-refactor-v2.draft.md`（draft 已评审，结论见 §3）
> **输入**: `validte-refactor.md`、`design-refactor-v2.draft.md`、`docs/TODO.md`、`CHANGELOG.md` 的 V2-TODO、当前源码（v1.5.7）

---

## 0. 本方案与 draft 的关系

draft（`design-refactor-v2.draft.md`）提出了正确的总方向——**缓存类型反射元数据 + 减少重复反射 + 实例池**，但其数据结构设计存在两处会导致返工的根本性问题（详见 §3）：

1. 把"类型级元数据"和"值级规则展开（slice/map 元素）"混在同一个 `StructMeta` 里缓存，而后者**本质上依赖具体值、不可按类型缓存**；
2. `ValidateContext` 在构造时即**预先**计算 `IsZero/IsEmpty/RValue`，对最常见的 `required`/map 路径反而增加开销。

本方案保留 draft 的目标与收益预期，重做核心数据结构，并补充 draft 遗漏的最大收益点：**规则字符串解析与参数类型转换也是类型级的，同样可缓存**。

---

## 0.1 实现状态对账（v1.6.0 落地后复审 · 2026-06-03）

> 本设计写于实现**前**。v1.6.0 已落地（10 个提交，as-built 记录见 `docs/plans/v1.6.0-implementation-plan.md`）。§1–§5 仍保留设计期的分析与方案原貌（有历史价值），但**实际实现有多处刻意偏离**——下表对账，偏离均为实现期更安全/更高收益的选择。

| 设计 § | 设计提案 | 实际实现（v1.6.0） | 偏离原因 |
|---|---|---|---|
| §4.1 分层 | 下沉 `internal/meta·fieldval·exec` | **全留根包**，`fieldValue`/`typeMeta` 不导出 | internal 迁移(P4-Part2)实测牵动 7+ 文件破 import 环、纯搬家零收益，**经决策不做** |
| §4.2 缓存 | per-field `RuleTpl` + **构建期参数类型转换** + 不可变 | `typeMeta.isStatic` 分类 + **零值快照整套规则模板**(`ruleTemplate`+`sync.Once`)，args 保持字符串、**运行期 `convertArgsType` 不变** | `%#v` 黄金测试无法验证 arg 类型转换正确性 + 自定义验证器/nil 语义风险；复用现有解析跑零值实例做快照 ⇒ 保证逐字节一致（P3b） |
| §4.2 缓存 | `typeKey{rt,tagVer}` + sync.Map + Implements 缓存 + FieldByIndex | ✅ **如设计**（cache.go，P2） | — |
| §4.3 FieldValue | `GetWithDefault` 返回 `*FieldValue`、**全管线流转**、消除源头 `Interface()` 装箱 | `fieldValue` **仅用于** `valueValidate`/`callValidatorValue` 去重第 2/3 次 `reflect.ValueOf`；`GetWithDefault` 仍返回 `any`，**源头 `Interface()` 装箱未消除** | 消除源头装箱需改 `DataFace.TryGet` 返回值 → 属破坏性 **v2.0**；P1 范围收窄为安全的内部去重 |
| §4.4 规则收集 | build(类型) / instantiate(值展开 DynamicFields) | 静态类型零值快照克隆；**动态类型完整走原 `parseRulesFromTag` 不变** | 动态路径保持原样最安全，黄金测试守护 |
| §4.5 管线 | 细粒度 `stage/check/buildArgIn` + fieldValue 流转 | **仅拆** `applyField` + `validateWildcardSlice`（粗粒度） | 细拆致热路径 **+2.22%**（Go 不内联，`-gcflags=-m` 确认）→ 回退 |
| §4.7 复用 | opt-in `Factory`+pool | Factory **延后未做**；实际最大杠杆是设计**未提及**的 **translator 共享 builtin 消息 + ctx 验证器懒构建** | 见下"设计盲点" |

**🔍 设计盲点（实现期 profiling 才发现的最大杠杆）**：本设计 §1 痛点分析聚焦"重复反射 + 未缓存解析"，但**漏掉了 per-Validation 构造成本**——P0 实测 `newEmpty()`=62 allocs 占扁平 struct 验证的 ~72%，其中 `NewTranslator()` 每实例拷贝 ~150 条 builtin 消息（13 allocs）、`newEmpty` 急切建 16 个 context 验证器绑定方法（~34 allocs）。P5a/P5b 针对性消除后 `newEmpty` 降到 **2 allocs**，是 v1.6.0 性能收益的主要来源（连 Map 验证都 -42% allocs）。**教训**：性能重构必须先 profiling 定位真实热点，而非仅凭代码审查推断。

**📊 实测累计（P0→终态，Go1.25，count=6）**：扁平 struct **-78% 耗时 / -74% 分配**；嵌套 -79%/-74%；Map -65%/-42%；首次解析 -91%/-78%。**远超**设计 §7 达标线（静态 ns↓≥50%、allocs↓≥40%）。顺带修了 2 个真实 bug：递归 struct 类型 `buildTypeMeta` 栈溢出、`Val/Var` 共享实例并发 race。

**⏳ 仍属 v2.0（破坏性，未做，见 §6.0）**：内置验证器统一 `reflect.Value` 入参、`DataFace` 扩展（**做了才能落地 §4.3 的"消除源头 `Interface()` 装箱"**）、注册自定义类型、清理 deprecated、`/v2`、Go 升 1.21。

---

## 1. 现状分析（基于真实代码精确定位）

> ⚠️ 本节为**设计期（v1.5.7）**现状快照；v1.6.0 后部分已变（如 §1.1 "cache.go 未实现" 已落地）。对账见 §0.1。

### 1.1 核心组件与现状

| 组件 | 文件 | 现状 |
|------|------|------|
| 入口 / 全局配置 | `validate.go` | `New/Map/Struct/Request`、`gOpt GlobalOption`、`FromStruct` 等 |
| 验证实例 | `validation.go` | `Validation` 结构（rules/errors/safeData/trans...） |
| 验证执行 | `validating.go` | `Rule.Apply`、`valueValidate`、`callValidator`、`callValidatorValue` |
| 数据源 | `data_source.go` | `DataFace` + `MapData`/`FormData`/`StructData` |
| 验证器 | `validators.go` | `funcMeta` + 70+ 内置验证器 + `IsEmpty` |
| 规则 | `rule.go` | `Rule` 结构 + `StringRule` 解析 |
| 全局注册 | `register.go` | `validators`/`validatorMetas`/`validatorAliases` |
| 工具 | `util.go` | 反射/类型转换/比较辅助 |
| 值校验 | `value.go` | `Val()` 直接校验值，共享 `emptyV` 实例 |
| 缓存 | `cache.go` | 设计期：全文注释未实现 → **v1.6.0 已实现**（typeMeta/fieldMeta + 静态规则模板，257 行） |

### 1.2 真实验证调用链

```
validate.Struct(&user)
  └─ FromStruct(s)            validate.go:343  反射取 value/typ，存 StructData
  └─ StructData.Create()      data_source.go:234
       └─ parseRulesFromTag() data_source.go:270  递归遍历【值树】解析 tag → v.StringRule(...)
  └─ v.Validate()             validating.go:49
       └─ for rule := range v.rules: rule.Apply(v)   validating.go:71
            └─ v.GetWithDefault(field)  → tryGet → StructData.TryGet()  data_source.go:530
            └─ r.valueValidate(field,name,val,v)      validating.go:250
                 └─ callValidator(...) / callValidatorValue(...)
```

### 1.3 痛点根因（逐条对应 `validte-refactor.md`）

#### 痛点 A：「很多地方重复使用反射」——同一个值被反射 2~3 次

对一个 struct 字段做一次校验，同一底层值经历的反射往返：

1. `StructData.TryGet` 命中缓存后 **`fv.Interface()`**（`data_source.go:539`）→ 把已有的 `reflect.Value` 拆箱成 `any`；
2. `valueValidate` 中 **`reflect.ValueOf(val)`**（`validating.go:300`）→ 把 `any` 重新装箱成 `reflect.Value`；
3. 自定义验证器走 `callValidatorValue` 时 **再次 `reflect.ValueOf(val)`**（`validating.go:549`）。

即 `reflect.Value → Interface() → reflect.ValueOf → reflect.ValueOf`。连最快的 `required` 路径也逃不掉：`v.Required` → `IsEmpty(val)` → `reflect.ValueOf(val)`（`validators.go:556`）。

> **根因**：值在管线中以 `any` 传递，每一层都要重新 `reflect.ValueOf` 还原类型信息。**缺少一个贯穿管线、携带 `reflect.Value` 的载体**。

#### 痛点 B：「指针 / 结构体 0 值场景判断复杂，传递时丢失原有类型」

指针解引用、零值判断散落在多处且各写各的：`removeValuePtr`（util.go:541）、`convValAsFuncValArgType` 里的指针处理（validating.go:377）、`callValidatorValue` 里的 `rftVal.Kind()==Ptr` 处理（validating.go:556）。`TryGet` 返回的 `zero` 与 `IsEmpty(val)` 是两套独立判断，且 `any` 化后 `*T(nil)` 与 `T` 零值难以区分。

> **根因**：同上——没有统一的「字段值对象」承载 `src/rv/rt/indirect/isZero/isEmpty`。`docs/TODO.md` 已经给出了正确雏形（`FieldValue`）。

#### 痛点 C：「没有缓存反射类型 / tags，性能不高」

每次 `validate.Struct(&sameType{...})` 都重新：
- 遍历整个值树（`parseRulesFromTag`，data_source.go:270），
- 逐字段 `fv.Tag.Get(...)` 解析 validate/filter/label/message/json 五种 tag，
- 逐条 `v.StringRule(name, rule)`：再做 `splitRules` + 别名解析（`ValidatorName`）+ `parseArgString` + `strings2Args`。

这些**全部是类型级的**（同一类型结果恒定），却每次实例都重算。`cache.go` 中早有 TODO 但未实现。

此外字段访问用 `FieldByName`（data_source.go:557/601），是 O(字段数) 的字符串匹配，而非 `FieldByIndex` 的 O(1)。

#### 痛点 D：「部分核心逻辑混乱，维护困难」

`valueValidate`（validating.go:250-371）单函数 120 行，混杂：safe 短路、`.*` 通配符 slice 展开（含多层 flatten）、map 通配符父切片长度对比、func 入参类型转换、内置/自定义验证器分派。`Rule.Apply`（validating.go:85-192）同样把 default/filter/skip-empty/validate/error 全部塞在一个循环里。

#### 痛点 E：「实现应下沉 internal，对外只留公共 type/interface/包级方法」

当前所有实现都在根包，公共面与内部实现耦合。`reflect.Value` 缓存、类型元数据、管线执行等纯内部机制暴露在包级。

### 1.4 反射热点清单（修正 draft 的行号）

| 位置 | 操作 | 频率 | 评级 |
|------|------|------|------|
| `data_source.go:539` | `fv.Interface()`（拆箱缓存值） | 每字段 | 🔴 |
| `validating.go:300` | `reflect.ValueOf(val)` | 每字段 | 🔴 |
| `validating.go:549` | `reflect.ValueOf(val)`（自定义验证器） | 每字段（自定义） | 🔴 |
| `validators.go:556` | `IsEmpty` 内 `reflect.ValueOf(val)` | 每 required 字段 | 🟡 |
| `validating.go:497/564/568` | `reflect.ValueOf(arg/data/args[i])` | 每验证器调用 | 🟡 |
| `data_source.go:557/601` | `FieldByName`（O(n)） | 每字段首访 | 🟡 |
| `data_source.go:270` | `parseRulesFromTag` 全量重解析 | 每实例 | 🔴 |

---

## 2. 重构目标与约束

### 2.1 目标

1. **类型元数据缓存**：同一 struct 类型的字段索引、tag、kind、**预解析规则模板**只构建一次。
2. **消除反射往返**：引入贯穿管线的 `FieldValue`，值在管线中携带 `reflect.Value`，杜绝 `Interface()↔ValueOf` 往返。
3. **统一指针 / 零值 / 空值语义**：集中到 `FieldValue` 的方法，移除散落逻辑。
4. **核心逻辑解耦**：拆分 `valueValidate`/`Apply` 为职责单一的小步骤。
5. **实现下沉 internal**：根包只留公共 API，反射/缓存/管线放 `internal/`。
6. **零行为回归**：现有全部测试（`issues_test.go` 等 5w+ 行用例）必须全绿。

### 2.2 关键约束（决策已确认）

> **发布策略（已确认，两步走）**：
> - **v1.6.0 = 本设计的 P0–P6 全部**，是**保持公共 API 兼容的内部性能重构**：不改 `validate.Struct/Map/Request/Val`、`Validation` 公共方法签名、内置验证器函数签名、struct tag 语义；模块路径仍为 `github.com/gookit/validate`。用户升级零改动，只是更快。
> - **v2.0.0 = 基于 1.6 再做破坏性精简**：落地 `CHANGELOG.md` 的 V2-TODO（验证器统一 `reflect.Value` 入参、`DataFace` 扩展、注册自定义类型、清理 deprecated、可能升 `/v2`）。详见 §6.0 发布路线图。

- **Go 版本（已确认）**：
  - v1.6.0 **维持 `go 1.19`**（go.mod 不动）——本设计无任何特性依赖 >1.19，守住低版本兼容。
  - v2.0.0 **升到 `go 1.21`**（已在项目 CI 矩阵内），换取 `slices`/`maps`/`min`/`max`/`clear`/`sync.OnceValue` 简化实现。
  - **跳过 1.20**：相对 1.19 仅多 `errors.Join`，对本重构无价值，且项目 CI 矩阵 `[1.19,1.21,1.22,1.23,1.24,1.25,stable]` 本就不测 1.20。
- 依赖：继续使用 `gookit/goutil/reflects`、`gookit/filter`。
- 并发：`validate` 被多 goroutine 并发使用（见 `BenchmarkFieldSuccessParallel`），缓存必须并发安全；共享的规则模板必须**不可变**。

### 2.3 预期收益

| 优化项 | 预期 |
|--------|------|
| 同类型重复验证（已缓存类型） | 反射调用 ↓ ~80%，耗时 ↓ 50-70% |
| 单字段验证内存分配 | 每字段少 1~2 次装箱 / `reflect.ValueOf` |
| 首次验证某类型 | 与现状持平或略升（多了缓存写入） |
| Map / Form 数据源 | 持平（无 struct 反射），但受益于管线简化 |

---

## 3. 对 draft 方案的评审结论

| draft 设计点 | 评审 | 说明 |
|---|---|---|
| 缓存 struct 类型元数据（TypeCache） | ✅ 采纳（重做结构） | 方向对，但需拆分类型级 / 值级，见 §3.1 |
| `StructMeta` 递归展平所有嵌套字段进一张表 | ❌ 否决 | slice/map 元素规则（`items.0.name`）依赖**值**，无法按类型缓存；draft 自己也注明"don't cache slice elements"，自相矛盾 |
| `FieldMeta` 字段（index/kind/tag/ptr...) | ✅ 大部分采纳 | 保留为 `internal` 结构，补 `[]int` 索引路径 |
| `ValidateContext` 携带反射值 | 🟡 修正 | 思路对，但**改为 `FieldValue` + 懒计算**，不在构造时算 IsZero/IsEmpty |
| 构造时即算 `RValue/IsZero/IsEmpty` | ❌ 否决 | 对 `required`/string 验证器是浪费；改懒求值 |
| Validation `sync.Pool` | 🟡 采纳但降级 | 改为**可选 `Factory`**，非默认行为；`Val()` 的共享 `emptyV` 需先解决并发 |
| 快速路径 switch 扩展 | ✅ 采纳 | 已有（validating.go:397），可适度扩充 |
| `BuildFieldPath` strings.Builder | ✅ 采纳 | 低风险小优化 |
| `MessageCache`（fmt.Sprintf 键缓存） | ❌ 否决 | 构造缓存键的 `fmt.Sprintf("%s:%v")` 与直接格式化等价开销，得不偿失 |
| 「规则解析每次重做」 | 🔼 draft 遗漏 | **本方案新增**：规则字符串解析 + 参数类型转换也是类型级，纳入模板缓存（最大增益之一） |

---

## 4. V2 核心设计

### 4.1 分层架构

```
github.com/gookit/validate                 (根包：薄公共层)
├── 公共 type/interface：Validation, DataFace, Rule, Errors, M/MS, GlobalOption
├── 包级入口：New/Map/Struct/Request/Val/Config/AddValidator...
└── 委托 internal 完成实际工作

github.com/gookit/validate/internal/
├── meta/        类型元数据缓存（TypeMeta / FieldMeta / RuleTpl）——纯类型级，可缓存
├── fieldval/    FieldValue：值级载体（src/rv/rt/real + 懒 isZero/isEmpty）
└── exec/        验证执行管线（步骤拆分）  [阶段4再下沉，先在根包内拆分]
```

> 渐进式：先在**根包内**完成结构拆分与缓存（降低 import 环依赖风险），稳定后再物理迁入 `internal/`。`internal/` 不影响公共 API。

### 4.2 `TypeMeta`：类型级元数据缓存（核心）

**唯一职责**：缓存"只与类型有关、与具体值无关"的信息。**绝不缓存 slice/map 元素的展开规则**。

```go
// internal/meta（示意，最终字段名以实现为准）

// TypeMeta 一个 struct 类型的全部类型级元数据
type TypeMeta struct {
    Type   reflect.Type
    Fields []*FieldMeta            // 仅直接字段 + 可静态展开的子 struct 字段
    byName map[string]*FieldMeta   // name/path -> meta，O(1)

    // 预解析、不可变的规则模板（关键增益：StringRule 解析只做一次）
    StaticRules []*RuleTpl         // 路径恒定的规则（leaf 字段、struct-of-struct）

    // 是否存在需要按值展开的字段（slice-of-struct / map-of-struct）
    HasDynamic  bool
    DynamicFields []*FieldMeta     // 需运行时遍历值来展开元素路径的字段

    // 类型级的翻译/标签信息（label、json 输出名）
    Labels  map[string]string
    Outputs map[string]string

    // 实现的可选接口（一次性判定，免每次 Implements）
    ImplConfig, ImplTranslates, ImplMessages bool
}

// FieldMeta 单字段类型级信息
type FieldMeta struct {
    Index   []int          // 用于 Value.FieldByIndex，O(1) 访问（替代 FieldByName）
    Name    string         // 字段名
    Path    string         // 点路径，如 "Parent.Child"
    Kind    reflect.Kind   // 已去指针后的 kind
    IsPtr   bool
    Elem    elemClass      // leaf / structKind / sliceOfStruct / mapOfStruct ...
    ValidateRule, FilterRule string
    OutputName, Label string
    Messages map[string]string
}

// RuleTpl 预解析、不可变的规则模板（构建时完成别名解析 + 参数类型转换）
type RuleTpl struct {
    Path       string        // 字段路径
    Validator  string        // 原始名
    RealName   string        // 别名解析后
    NotRequired bool
    Args       []any         // 已按验证器签名转换好的不可变参数
    FuncMeta   *funcMeta     // 全局/内置验证器可在构建期绑定；struct 方法验证器留空、运行时解析
    FilterRule string
}
```

**缓存容器**（替代 draft 的手写 RWMutex + double-check）：

```go
// 用 sync.Map，读多写少；key 含 tag 配置版本号，解决配置变更失效（见 §4.6）
var typeCache sync.Map // map[typeKey]*TypeMeta

type typeKey struct {
    rt      reflect.Type
    tagVer  uint32  // gOpt tag 配置变更时自增
}
```

**关键设计决策**：
- `StaticRules` 在构建期就完成 `splitRules` / `ValidatorName` 别名解析 / `parseArgString` / 参数类型转换（`convertArgsType` 中依赖的是 *验证器签名 + 字面参数*，二者皆类型级），运行时只需轻量克隆/引用。**这是 draft 完全遗漏的最大增益点。**
- 参数转换结果存入 `RuleTpl.Args` 后**不可变**；运行时禁止再 in-place 改写（现状 `validating.go:530` 的 `args[i]=...` 必须移除，改为构建期完成）。
- 仅 struct-of-struct 可静态展开进 `StaticRules`；slice/map-of-struct 标记到 `DynamicFields`，运行时按值遍历展开（见 §4.4）。

### 4.3 `FieldValue`：值级载体（消除反射往返）

落实 `docs/TODO.md` 的 `FieldValue` 雏形，并补齐懒求值与指针/零值语义。

```go
// internal/fieldval

type FieldValue struct {
    name string         // 字段名/路径
    src  any            // 原始值（来自 data source）
    rv   reflect.Value  // = reflect.ValueOf(src)，懒初始化
    rt   reflect.Type   // = rv.Type()，懒
    real reflect.Value  // = reflect.Indirect(rv)，去指针后，懒
    meta *FieldMeta     // 可空（map/form 源无类型元数据）

    rvInit   bool       // rv/rt 是否已初始化
    realInit bool
    zero     int8       // 0=未知 1=zero 2=notzero（懒）
    empty    int8       // 0=未知 1=empty 2=notEmpty（懒）
}
```

要点：
- **来源即载体**：`StructData` 取字段时直接构造 `FieldValue` 并把已有的 `reflect.Value` 塞进 `rv`（**不再 `Interface()` 拆箱**），杜绝痛点 A 的第 1、2 跳。
- **懒求值**：`RV()/Real()/IsZero()/IsEmpty()` 按需计算并缓存到结构体；`required`/string 验证器路径可能完全不触发反射。
- **指针/零值集中**：`Real()` 统一去指针；`IsZero/IsEmpty` 单点实现，删除 `convValAsFuncValArgType`、`callValidatorValue` 中分散的指针处理。
- **map/form 源**：无源 `reflect.Value`，`src` 为普通 `any`，`rv` 在首个需要反射的验证器处懒构造——与现状等价，无额外成本。

`Validation.GetWithDefault` 改为返回 `*FieldValue`（内部），公共 `Get/Safe` 等签名不变（内部 `.src`）。

### 4.4 规则收集重构：类型级模板 + 值级元素展开分离

把现 `parseRulesFromTag`（值树遍历 + tag 解析 + StringRule）拆成两段：

```
build(rt) ──► TypeMeta{StaticRules, DynamicFields, Labels...}   // 仅一次，进缓存
                       │
instantiate(value) ────┤
                       ├─ 直接挂载 StaticRules（零反射）
                       └─ 对 DynamicFields 遍历【真实值】展开元素规则
                            items.0.name / items.1.name / m."k".age ...
```

- **构建期 `build`**：只遍历**类型**（`Type.Field(i)` 递归 struct），解析全部 tag，生成 `StaticRules` 与翻译表；遇到 slice/map-of-struct 记入 `DynamicFields`（记录元素类型的子 `TypeMeta`，递归缓存）。
- **实例期 `instantiate`**：常见的「扁平 struct / struct 套 struct」**完全命中缓存、零反射**；只有真正含 slice/map-of-struct 时才按值遍历（此部分逻辑从现 `parseRulesFromTag` 的 `case Array/Slice/Map` 平移，保持行为一致，含 `CheckSubOnParentMarked`、`IsSimpleKind` 跳过、nil 指针初始化等现有分支）。

> 兼容关键：现状对「父字段无规则则跳过子 struct」（`CheckSubOnParentMarked`，data_source.go:359）、「nil 指针字段且有规则则初始化」（data_source.go:430）等行为必须在两段式中**逐一保留**，用回归测试守护。

### 4.5 验证执行管线重构（拆分巨函数）

把 `Rule.Apply` + `valueValidate` 拆为线性、可读的步骤，统一以 `*FieldValue` 流转：

```go
func (r *Rule) Apply(v *Validation) (stop bool) {
    for _, field := range r.fields {
        if v.skipByScene(field) { continue }
        fv := v.fieldValue(field)              // 构造/复用 FieldValue（携带 rv）

        switch r.stage(fv) {                    // default / filter / skip-empty 前置
        case stageSkip:     continue
        case stageStop:     return true
        }
        if r.check(fv) {                        // 仅做"取验证器并调用"
            v.saveSafe(fv)
        } else {
            v.addError(fv, r)
        }
        if v.shouldStop() { return true }
    }
    return false
}
```

`check(fv)` 内部再分三个明确分支（各自独立函数，便于测试）：
1. `RuleSafe`/`-` 短路；
2. `required*` 走快速直调（`v.Required` 等，传 `fv` 免再反射）；
3. 通配符 `.*` slice 元素校验（`flatSlice` 多层展开逻辑平移）；
4. 普通验证器：内置走 `callValidator` 的 switch 快速路径；自定义走反射调用，但**复用 `fv.RV()`**，不再 `reflect.ValueOf(val)`。

验证器入参构造统一封装 `buildArgIn(fv, tpl)`，`data`/`val` 位用 `fv.RV()`，`args` 用模板里**已转换好**的不可变参数，消除每次调用的 `reflect.ValueOf(arg)`（现 validating.go:497/568）。

### 4.6 缓存失效与并发安全

| 场景 | 策略 |
|---|---|
| 运行时 `Config()` 改 tag 名（ValidateTag 等） | `Config()` 内自增 `tagVer`；`typeKey{rt,tagVer}` 自然失效旧缓存（不清空、不加锁全表） |
| `ResetOption()` | 同上，`tagVer++` |
| 全局 `AddValidator` 运行时注册 | 不影响 `TypeMeta`（缓存类型，不缓存验证器集合）；`RuleTpl.FuncMeta` 对未知名留空、运行时解析 |
| struct 方法验证器（`StructData.FuncValue`） | 实例级，`RuleTpl` 不绑定，运行时解析（同现状回退路径 validation.go:243） |
| `*T` 与 `T` | 分别为不同 `reflect.Type`，`build` 内对入口统一 `reflects.Elem` 取 elem 类型后再缓存，二者命中同一 elem 类型缓存 |
| 并发 | `sync.Map` 读写；`TypeMeta`/`RuleTpl` 构建完成后**只读不可变**；实例期只读模板、写自身 `Validation` |
| 缓存无界增长 | 默认不清理（类型数有限，draft §5.2.6 评估 ~200KB 量级可接受）；提供 `validate.ResetTypeCache()` 公共方法手动清空 |

### 4.7 Validation 复用（可选，非默认）

draft 想用 `sync.Pool` 包住默认 `NewValidation`，风险高（`Reset` 不彻底易串数据、`Val()` 共享 `emptyV`）。本方案降级为**显式可选**：

```go
// 可选工厂：适合"同类型大批量校验"的热路径，由用户显式选用
f := validate.NewFactory()
v := f.Struct(&user)   // 内部 pool 取 Validation + 命中 TypeMeta 缓存
defer v.Release()      // 归还前 Reset
v.Validate()
```

- 默认 `validate.Struct/Map` 行为、生命周期**完全不变**。
- `value.go` 的共享 `emptyV` 改造：要么每次 `Val()` 用轻量栈对象，要么从独立 pool 取，**消除现有并发隐患**（当前 `emptyV` 全局共享，`Val` 并发写 `es`/rule 实际上各自栈分配，但 rule.valueValidate 复用 emptyV.validatorMetas 只读尚可——重构时一并明确）。

---

## 5. 兼容性策略

✅ **保持不变**
- 包级入口：`New/Map/JSON/Struct/Request/Val/Var`、`Config/Option/ResetOption`、`AddValidator(s)`。
- `Validation` 公共方法：`Validate/ValidateE/ValidateErr`、`StringRule(s)/AddRule/AppendRule`、`Get/Safe/SafeData/BindStruct`、`AtScene/WithScenes/WithMessages/WithTranslates` 等。
- struct tag 语义：`validate/filter/label/message/json/default`。
- `DataFace` 接口与 `MapData/FormData/StructData` 公共字段。

⚠️ **内部变化（用户无感）**
- `StructData` 内部：`fieldValues map[string]reflect.Value` → 由 `TypeMeta` + `FieldValue` 取代；保留 `fieldNames` 兼容（或内部映射）。
- `Rule.arguments` 不再运行时 in-place 转换。
- `parseRulesFromTag` 行为拆成 build/instantiate（输出规则集合需逐字段比对一致）。

🔶 **需重点回归的语义**（编写专项用例锁定）
- 指针字段 / nil 指针自动初始化（data_source.go:430-439）。
- 零值与 `required` 的交互（`isIgnoreableZeroNumeric`、`CheckZero`）。
- `.*` 通配符多层 slice、map 通配符父切片长度对比（validating.go:303-358, getParentSliceLen）。
- 匿名/私有字段（`ValidatePrivateFields`、`fieldAtAnonymous`，data_source.go:290/598）。
- `CheckSubOnParentMarked` 子结构按需收集。

---

## 6. 实施计划

### 6.0 发布路线图（两步走，已确认）

本设计与 `CHANGELOG.md` 的 V2-TODO 合并后，按"是否破坏公共 API"切成两个版本：

| 项目 | 来源 | 是否破坏 API | 归属版本 |
|---|---|---|---|
| TypeMeta 类型元数据缓存（§4.2） | 本设计 | 否 | **v1.6.0** |
| FieldValue 值载体、消除反射往返（§4.3） | 本设计 / TODO.md | 否 | **v1.6.0** |
| RuleTpl 规则模板缓存（§4.4） | 本设计 | 否 | **v1.6.0** |
| 验证管线拆分、实现下沉 internal（§4.5/§4.1） | 本设计 | 否 | **v1.6.0** |
| 可选 Factory + sync.Pool（§4.7） | CHANGELOG V2-TODO | 否（opt-in） | **v1.6.0** |
| 内置验证器统一 `reflect.Value` 入参（`Gt(val any,dst int64)`→`gt(reflect.Value,...)`） | CHANGELOG V2-TODO | **是** | **v2.0.0** |
| `DataFace` 扩展（`BindStruct/SafeVal` 进接口） | CHANGELOG V2-TODO | **是** | **v2.0.0** |
| 注册自定义类型（can register custom type） | CHANGELOG V2-TODO | 是 | **v2.0.0** |
| 清理 deprecated（`ValueLen` 等）、`/v2` 模块路径、Go 升 1.21 | 本设计 | 是 | **v2.0.0** |

**两步走的协同点（需知晓的"先建后拆"）**：v1.6.0 为不改验证器签名，`RuleTpl` 在构建期**预转换参数类型**（验证器要 `int64/int` 等具体类型）。到 v2.0.0 验证器统一收 `reflect.Value` 后，`convertArgsType`（validating.go:472）这套按调用转换的机器可大幅删除——属顺势简化，非返工。v1.6.0 期间这套预转换实打实在提速。

> 本章 §6.1 的 P0–P6 即 **v1.6.0** 的落地阶段。v2.0.0 的破坏性内容待 1.6.0 稳定发布后另立设计文档。

### 6.1 v1.6.0 分阶段实施

> 原则（遵循 AGENTS.md）：**每个子阶段独立可合并、独立提交、benchmark 不回退**；改动跨 >3 文件或 >100 行业务代码前与用户确认。每阶段以「全部现有测试通过 + 基准不劣化」为门禁。

| 阶段 | 内容 | 产出 | 风险 | 门禁 |
|---|---|---|---|---|
| **P0 基线** | 补齐基准：扁平 struct / 嵌套 struct / slice-of-struct / map / Val()；记录 ns/op、allocs/op | `benchmark_test.go` 扩充 | 低 | 基线数据存档 |
| **P1 FieldValue** | 落地 `FieldValue`（懒求值 + 指针/零值集中）；先在 `valueValidate`/`callValidatorValue` 内部用它替换重复 `reflect.ValueOf`，**不改对外行为** | `fieldval` 雏形 | 低 | 测试全绿、单字段 allocs ↓ |
| **P2 类型元数据缓存** | `TypeMeta/FieldMeta` 构建 + `sync.Map` 缓存 + `tagVer` 失效；`StructData.TryGet` 改用 `FieldByIndex` | `meta` 雏形 + 缓存 | 中 | 同类型二次验证 ns/op 明显下降 |
| **P3 规则模板缓存** | `RuleTpl`：构建期完成别名解析 + 参数转换；`Create()` 走 build/instantiate 两段式 | 规则模板 | 中 | 规则集合与旧实现逐字段一致（diff 测试） |
| **P4 管线拆分** | 拆 `Rule.Apply`/`valueValidate` 为小步骤；统一 `FieldValue` 流转；下沉 `internal/` | 可读管线 | 中 | 行为零回归 |
| **P5 可选工厂/池** | `Factory` + `Release`；改造 `emptyV` 并发安全 | 可选优化 | 低 | 并发/内存泄漏测试 |
| **P6 收尾** | 文档、迁移说明、`docs/TODO.md` 的 `notIn`/`unique` 等新验证器（独立 PR） | 文档 | 低 | 覆盖率 ≥ 现状 |

提交规范示例：`refactor(meta): add TypeMeta cache for struct reflect metadata`、`perf(fieldval): reuse reflect.Value to remove round-trip`。

---

## 7. 测试与基准策略

**基准（`-benchmem` 必看 allocs/op）**
```go
func BenchmarkStructFlat(b *testing.B)        // 3 字段，重复验证同类型
func BenchmarkStructNested(b *testing.B)      // struct 套 struct
func BenchmarkStructSliceOfStruct(b *testing.B)
func BenchmarkMapValidate(b *testing.B)       // 应持平
func BenchmarkValValue(b *testing.B)          // validate.Val()
```
对照表（达标线）：

| 场景 | 现状(基线) | v2 目标 |
|---|---|---|
| 扁平 struct 重复验证 | P0 实测 | ↓ ≥50%，allocs ↓ ≥40% |
| 嵌套 struct | P0 实测 | ↓ ≥50% |
| Map 验证 | P0 实测 | 持平（±5%） |
| 首次验证新类型 | P0 实测 | ≤ 基线 +15% |

**正确性**
- 全量跑 `issues_test.go`、`issues_x2_test.go`、`validation_test.go`、`validators_test.go`、`morecase_test.go`、`data_source_test.go`——零回归。
- 新增 `cache_test.go`：缓存命中（`sync.Map` 返回同实例）、`tagVer` 失效、`*T`/`T` 共享、并发构建无竞态（`-race`）。
- 新增 `regression_compat_test.go`：同一组 struct，**对比 v2 生成的规则集合与 v1 一致**（path/validator/args 全等），守护 build/instantiate 拆分。
- `-race` 跑并发基准。

断言库：根 `AGENTS.md` 规范为 `github.com/gookit/goutil/x/assert`，而 validate 现有测试用的是 `github.com/gookit/goutil/testutil/assert`（两者 API 基本一致）。**新增测试统一采用 `x/assert`**；存量测试可在阶段收尾时按需迁移，不强制本次完成。多用例用 `t.Run` 分组。

---

## 8. 风险与权衡

| 风险 | 影响 | 缓解 |
|---|---|---|
| build/instantiate 拆分遗漏某分支，规则集合不一致 | 高 | P3 引入 v1/v2 规则 diff 测试；分支逐条核对（指针初始化、CheckSubOnParentMarked、私有字段） |
| `FieldByIndex` 对匿名/私有字段的可访问性与 `FieldByName` 差异 | 中 | 保留 `fieldNames` 分类（top/anonymous/sub），用例覆盖 `ValidatePrivateFields` |
| `RuleTpl.Args` 不可变后，依赖运行时 in-place 转换的隐式行为变化 | 中 | 参数转换前移到构建期，输出与旧逻辑比对 |
| 缓存的 `funcMeta` 绑定全局验证器，运行时 `AddValidator` 覆盖同名 | 低 | 模板仅对**构建时已存在**的内置/全局验证器绑定；自定义/同名覆盖走运行时解析 |
| `internal/` 物理迁移引发 import 环 | 中 | 先在根包内完成拆分（P1-P4），P4 末尾再迁移；保持单向依赖 meta←fieldval←exec |
| Validation 池化数据串扰 | 中 | 池化设为**显式可选**，非默认；`Release` 前 `Reset` 全字段；`-race` + 内存检测 |

---

## 9. 附录

### 9.1 文件改动清单

| 文件 | 改动 | 阶段 |
|---|---|---|
| `cache.go` | 实现 `TypeMeta` 缓存（替换注释） | P2 |
| `internal/meta/*.go`(新) | `TypeMeta/FieldMeta/RuleTpl` + build | P2/P3 |
| `internal/fieldval/*.go`(新) | `FieldValue` | P1 |
| `data_source.go` | `StructData` 改用缓存 + `FieldByIndex`；`parseRulesFromTag` 拆 build/instantiate | P2/P3 |
| `validating.go` | 管线拆分，`FieldValue` 流转，复用 `rv` | P1/P4 |
| `rule.go` | `RuleTpl` 克隆/引用，参数不可变 | P3 |
| `validation.go` | `fieldValue()`、`Factory/Release`（可选） | P1/P5 |
| `value.go` | `emptyV` 并发安全改造 | P5 |
| `validate.go` | `Config()` 自增 `tagVer`；`ResetTypeCache()` | P2 |
| `*_test.go` | 基准扩充、缓存测试、v1/v2 规则 diff | 全程 |

### 9.2 与 `docs/TODO.md` 的对齐

- ✅ `use pool for Validation` → §4.7（降级为可选 Factory）
- ✅ `cache struct reflect type, tags` → §4.2 `TypeMeta`
- ✅ `field value struct` → §4.3 `FieldValue`（直接采用 TODO 的 `name/src/rv/rt/real` 雏形）
- ✅ `实现下沉 internal` → §4.1
- 🔜 新增验证器 `notIn`(已存在?)/`unique`/`distinct`/`dateFormat`/`dateEquals`/`beforeDate`... → **独立功能 PR**，不纳入本次性能重构（避免混合变更）

### 9.3 术语

| 术语 | 含义 |
|---|---|
| TypeMeta | struct 类型级、可缓存的反射元数据（字段/标签/静态规则模板） |
| FieldMeta | 单字段类型级信息（含 `[]int` 索引路径） |
| RuleTpl | 预解析、不可变的规则模板（别名/参数已就绪） |
| FieldValue | 值级载体，携带 `reflect.Value`，懒求值，贯穿管线 |
| StaticRules / DynamicFields | 路径恒定的规则 / 需按值展开的 slice·map-of-struct 字段 |
| tagVer | tag 配置版本号，用于缓存失效 |

---

## 10. 决策记录（已确认）

| # | 决策点 | 结论 |
|---|---|---|
| 1 | **兼容级别 / 发布节奏** | ✅ **两步走**：v1.6.0 做 API 兼容的内部性能重构（P0–P6），1.6 稳定后基于它做破坏性 v2.0.0（落地 CHANGELOG V2-TODO，丢历史包袱）。映射见 §6.0。 |
| 2 | **新验证器**（`unique/distinct/dateFormat...`） | ✅ **暂不做，后续独立 PR**。注：`notIn` 已存在（register.go:43），TODO 可划掉。 |
| 3 | **Go 版本** | ✅ **v1.6.0 维持 1.19**（守低版本兼容）；**v2.0.0 升 1.21**；**跳过 1.20**（无价值且 CI 不测）。 |

---

**文档结束** — 决策已确认，下一步从 **§6.1 P0 基线基准** 开始实施 v1.6.0。
