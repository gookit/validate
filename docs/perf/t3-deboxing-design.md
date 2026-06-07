# T3 专项设计子稿：去 `TryGet` 的 `fv.Interface()` 装箱

> **类型**：设计/评估稿（路线图 S4 / §3.4 P3a）。**未改任何业务代码。**
> **日期**：2026-06-07
> **前置**：`v2.x-perf-optimization-plan.md`（S1/S2/S3/S3.1 已落地，`Check` 池化 = 6 allocs）。
> **范围**：仅 **struct 数据源**（map/form 源的值本就是 `any`，不经反射装箱）。

---

## 0. 结论先行（TL;DR）

**经数据流分析，T3 在 gookit 的"收集结果"主路径上收益 ≈ 0，建议不实施。** 原因一句话：

> `Check`/`ValidateR` 校验通过的每个字段都会被存入 `safeData map[string]any`（供 `SafeData()`/`BindStruct`），存入时**必须**把字段值装箱成 `any`。而 `TryGet` 的那次装箱**正好被复用**为 safeData 的值（`tryGet` 先读 `safeData` 命中即返回，见 §2）。所以"装箱"在收集路径上是 **safeData 输出的固有成本**，不是可以靠"端到端传 reflect.Value"省掉的浪费。

- T3 的设想是"全程携带 `reflect.Value`，只在校验器真要 `any` 时才装箱"。但**几乎所有内建校验器都是 `any`-原生**（`Min/Max/Lt/Gt/Enum/IsInt/Length/valToString…`，§3），要去装箱必须**重写整个校验器分发面**（~70 个 case + `reflectx.ValueCompare` 等底层），golden 风险极高、改动面巨大。
- 即便重写到位，通过的字段仍要为 safeData 装箱一次 → **净收益约为 0**（仅"被取值但既不入 safeData 也不需字符串"的边角字段能省，属错误/跳过路径，见 §4）。
- **真正能逼近 go-playground 0 alloc 的杠杆不是去装箱，而是"不收集 safeData"**——即新增一个 opt-in 的纯过/败入口（如 `CheckErr(s) error`），跳过 safeData/filteredData/ValidResult。这是**功能取舍**，与 T3 是两回事，需单独评估（§6）。

**建议**：S4 关闭"去装箱"方向；剩余 6 allocs 视为 gookit"收集校验结果"的设计成本。若业务确有 sub-6 的极致诉求，按 §6 先 spike `CheckErr` 原型量化，再决定。

---

## 1. 现状：`Check` 池化路径的 6 个 allocs（memprofile 制导）

profile：`tmp/check_mem.prof`（`BenchmarkGookitCheckValid`，flat struct 3 字段）。按 `alloc_objects` 拆解：

| # | 来源 | 个数 | 性质 |
|---|---|---|---|
| 1 | `reflect.packEface`（`TryGet` 的 `fv.Interface()` 装箱） | ~3 | 本稿目标 |
| 2 | safeData map：`ensureSafeData` 头 + `applyField:247` 插入桶 | 2 | **必需输出** |
| 3 | `ValidResult` 壳（`ValidateR`） | 1 | **必需** |

> 注：装箱个数与字段类型相关——**Go runtime 对 0–255 的小整数有装箱缓存**（`staticuint64s`），故 `Age=30` 装箱免分配；`Name`/`Email`（string，2 字长）装箱各 1 alloc。即装箱 alloc ≈ "非缓存类型字段数"。

---

## 2. 数据流详解（值取得 → 校验 → 存储）

代码坐标（struct 源、单字段单规则）：

```
applyField (validating.go:164)
  └─ val, exist, isDefault := v.GetWithDefault(field)         // :185  —— 取值
        └─ v.tryGet(field) (validation.go:565)
              ├─ if val,ok := v.filteredData[key]; ok → 返回   // 命中过滤数据
              ├─ if val,ok := v.safeData[key];    ok → 返回   // ★命中已校验数据(复用装箱)
              └─ v.data.TryGet(field) (data_source.go:359)
                    └─ return fv.Interface(), ...              // ★★ 装箱发生处(struct 源)
  └─ r.valueValidate(field, name, val, v)                     // :242  —— 校验
        ├─ if name==required && !dotStar → v.Required(field,val)   // 最常用快路径(:344)
        ├─ fv := fieldval.New(field, val); rftVal := fv.RV()       // :394 RV()=reflect.ValueOf(val), 不装箱
        └─ callValidator(v, fm, field, val, …)                     // :420 → switch 消费 val any(:532)
  └─ if 通过 { v.ensureSafeData(); v.safeData[field] = val }   // :246-247 —— ★★★ 存储(复用同一 any)
```

**关键链路事实**：

1. **装箱只发生在 struct 源的 `StructData.TryGet` → `fv.Interface()`**（data_source.go:373/463 等）。map/form 源的值本来就是 `any`，无此装箱。
2. **同一字段多条规则只装箱一次**：第 1 条规则通过后 `safeData[field]=val`；后续规则的 `tryGet` 在 `safeData` 命中（validation.go:573 一带）直接返回同一个 `any`，**不再装箱**。即当前已是"≈1 装箱/已存字段"。
3. **存储 `safeData[field]=val` 复用的就是 `TryGet` 那次装箱的 `any`**——没有第二次装箱。

---

## 3. 障碍：校验器是 `any`-原生的

`callValidator`（validating.go:532）的 ~70 个 case 几乎全部消费 `val any`：

- 比较类：`Min/Max/Lt/Gt/Between/Enum/NotIn` → `reflectx.ValueCompare(val, …)`（validators_compare.go:114 等），**入参 `any`，内部再 `reflect.ValueOf`**。
- 类型类：`IsInt(val any)`/`IsString(val any)`/`IsNumber/IsNumeric/IsSlice` → 入参 `any`。
- 字符串类：`valToString(val any)` 取串后 `IsEmail(s)/Regexp/IsJSON/IsURL…`。
- `required*`：`v.Required(field, val any)`、`v.RequiredIf(field, val any, …)`。

> 要"端到端传 `reflect.Value`、只在必要时装箱"，必须给这些函数加 `reflect.Value`（或携带 RV 的 `fieldval.FieldValue`）变体、并改 `reflectx.ValueCompare`/`valToString` 等底层。`fieldval` 已有 `NewRV(name, rv)`（internal/fieldval/fieldval.go），但它 `f.Src = rv.Interface()` 仍**急切装箱**——也要改。这是一个跨 `validating.go` + `validators_*.go` + `internal/reflectx` + `internal/fieldval` 的大面改造。

---

## 4. 核心论证：为什么 T3 在收集路径上收益 ≈ 0

设字段 F 校验通过：
- **现状**：`TryGet` 装箱 1 次 → `val any` → 校验器消费 → `safeData[F]=val`（复用同一 any）。**净 1 装箱**，且该装箱是 safeData 必需的。
- **T3 后**（假设校验器全 RV 化）：取 RV（不装箱）→ RV 校验 → `safeData[F]=?` 需要 `any` → **此处必须装箱 1 次**。**仍净 1 装箱**。

→ **对"通过且入 safeData"的字段，装箱次数不变**（只是从 `TryGet` 挪到 `safeData` 存储处）。这正是收集结果型 API 与 go-playground（只返回 error、不收集）的固有差。

T3 唯一能省装箱的字段是**取了值但既不入 safeData、也不需要字符串/any 的**：
- `skipEmpty` 且值为空 → 在 `applyField:237` 跳过校验，但 `:185` 已取值装箱（可省）。
- 首条规则即失败、且该字段此前未被任何通过规则写入 safeData（注意：带 `required` 时 required 通常先通过并写 safeData，故此情形多见于**无 required** 的字段在错误路径）。

这些都属**错误/跳过路径**，非性能热点；且省下的也仅是非缓存类型字段那一两个 alloc。**性价比极低、风险极高。**

---

## 5. 若仍要实施 T3：改造点逐一列举 + golden 风险

（仅供完整性参考；不建议执行。每点都需逐字节 golden + race。）

| # | 改造点 | 文件/位置 | 内容 | golden 风险 |
|---|---|---|---|---|
| T3-1 | `tryGetRV` 内部取值 | `data_source.go StructData.TryGet` | 增不装箱的 reflect.Value 返回路径，保留 `Interface()` 兜底 | 中：sub-path/wildcard/指针叶子(#217)/`CanInterface` 分支必须等价 |
| T3-2 | 取值链贯通 RV | `validation.go tryGet/GetWithDefault`、`applyField` | 让 `val` 改携带 `fieldval.FieldValue`(含 RV)而非裸 `any`；defaultValue/filter 分支同步 | **高**：filteredData/safeData 命中分支返回的是 `any`，要与 RV 路径合流 |
| T3-3 | `fieldval` 真 RV 化 | `internal/fieldval/fieldval.go` | `NewRV` 不再急切 `Src=rv.Interface()`；`Src` 改懒装箱 | 中：所有读 `fv.Src` 处需走懒装箱 |
| T3-4 | 校验器 RV 变体 | `validating.go callValidator` + `validators_*.go` | 为 `Min/Max/Lt/Gt/Between/Enum/IsInt/IsString/Length/valToString…` 加 RV 入参路径 | **极高**：~70 case + 比较/取串语义，任一类型转换差异即破 golden |
| T3-5 | 底层比较/取串 RV 化 | `internal/reflectx.ValueCompare`、`valToString` | 加 `reflect.Value` 版，避免内部二次 `reflect.ValueOf` | 高：命名类型/指针/可转换值的等价 |
| T3-6 | safeData 存储装箱点 | `applyField:247` | 通过字段在此处 `fv.Interface()` 装箱入 safeData | —（必需，省不掉，见 §4） |
| T3-7 | wildcard slice / custom | `validateWildcardSlice`、自定义校验器入参 | 子元素仍 `Interface()`；自定义校验器签名是 `any` 必装箱 | 高：#283/#324 等用例 |

**分步与回滚**：即便做，也须 T3-1 → T3-3 → 单个标量校验器试点（如仅 `min/max`）→ 逐类扩展，每步独立 commit + 逐字节 golden，可单独 revert。预计十余次提交、数百行热路径改动，换取错误/跳过路径上 1–2 个 alloc。

---

## 6. 备选（真正有效的杠杆）：opt-in "不收集 safeData" 入口

若业务确有"高频纯校验、不取清洗数据"的诉求，**不收集结果**才是逼近 go-playground 的正道：

- 设想：`func CheckErr(structPtr any, scene ...string) error`（或 `v.ValidateErr` 的池化版），内部用包级池实例，**跳过 `safeData`/`filteredData` 的收集与 `ValidResult` 壳**，只跑过/败并返回 `Errors.OneError()`。
- 预估天花板（基于 §1 拆解）：去掉 safeData(2) + ValidResult 壳(1) ≈ **6 → ~3 allocs**；剩余 ~3 是装箱——而**不收集 safeData 后，装箱可真正去掉**（值不再需要存成 any），配合 §5 的 RV 校验器可进一步压到接近 0。即"不收集"是让 T3 变得**有意义**的前提。
- 代价：新 API 面 + 文档；`required` 之外的取值仍要喂校验器（部分仍需 `any`/string）。**仍是大工程**，但方向正确（直击固有成本而非搬运它）。

> ⚠️ 注意 §2 事实 2：safeData 当前**兼任"同字段多规则的装箱去重缓存"**。`CheckErr` 去掉 safeData 后，多规则字段会重复装箱/取值——需用一个**仅本次校验生命周期内的轻量 per-field 缓存**（栈上或小数组，非 map）补偿，否则装箱不降反升。这点要在 spike 里实测。

**建议**：先用一个**抛弃式 spike**（不进主干）量化 `CheckErr` 的真实 allocs 与对 multi-rule 字段的影响，再决定是否落地。spike 产物存 `tmp/`。

### 6.1 Spike 实测结果（2026-06-07，已验证，代码已回退）

抛弃式原型测了 3 个变体（bench `gookitFlat`：Name 3 规则/Email 2/Age 3；报告存 `tmp/spike-checkerr-report.md`）：

| 变体 | 入口 | ns/op | B/op | allocs/op |
|---|---|---:|---:|---:|
| A 基线 | `Check`（收集 safeData + 建 ValidResult） | 1389 | 405 | **6** |
| B 无去重 | `CheckErr`（跳过 safeData/filtered/壳，**不**装箱去重） | 1428 | 105 | **8 ⚠️负优化** |
| C 1槽去重 | `CheckErr` + 连续同字段取值去重 | 1175 | 40 | **3** |

**关键证实**：

- **B 反升到 8 allocs（比基线 +2）**：跳过 safeData 后丢失"装箱去重"——8 条规则各自 `TryGet→fv.Interface()` 重新装箱（约 8 次 vs 基线 3 次），多出的 ~5 次装箱把省下的 3 个（safeData 2 + 壳 1）全部抵消并反超。**单跳过 safeData 是负优化**，§4/§6 的预判成立。
- **C 降到 3 allocs**（−50% allocs / B/op −90% / ns −16%）：补上"连续同字段 1 槽去重"把每字段装箱压回 1 次，才兑现"省 3 个"的理论值。
- **1 槽够用前提**：`gookitFlat` 同字段多规则在 `v.rules` 中连续，故 1 槽命中率接近理想；若规则编排不保证同字段连续则退化回 B。正式化需：去重机制与 skipCollect 解耦、处理 zero 标志（`SkipOnEmpty`/默认值依赖）、补失败路径基准（`OneError`/`errorx.Raw` 的 alloc）。

**净结论**：`CheckErr` 形态 C **确实能把"只要过/败"的入口压到 3 allocs**，但**必须连同装箱去重一起做**，否则负优化。是否落地是**产品 API 决策**（值不值得为一个 opt-in 快速入口加这套机制 + 维护成本），不再是 T3"去装箱"那种纯性能改造。

---

## 7. 决策建议

1. **S4 / T3（去装箱）：不实施。** 收益≈0（§4），风险/改动面极大（§3/§5）。在路线图中标注"已评估，关闭"。
2. **当前 6 allocs 视为 gookit 收集校验结果的设计成本**，已远优于起点（flat-valid 23→6 Check / 23→12 非池化），且全程行为逐字节不变。
3. **若未来确有 sub-6 诉求**：走 §6 的 `CheckErr`（不收集 safeData）方向，先 spike 量化，**不要**走"端到端去装箱"。

---

## 8. 附：本稿涉及的代码坐标

| 用途 | 位置 |
|---|---|
| 装箱发生处（struct 源） | `data_source.go StructData.TryGet` → `fv.Interface()` |
| 取值链 | `validation.go tryGet/GetWithDefault`；`applyField`(`validating.go:164`) |
| safeData 命中复用 / 存储 | `validation.go tryGet`（读 safeData）；`validating.go:246-247`（写 safeData） |
| 校验分发 | `validating.go valueValidate:325` / `callValidator:532` |
| any-原生校验器 | `validators_compare.go`(Min/Max/…→`reflectx.ValueCompare`)、`validators_type.go`(IsInt/IsString)、`valToString`(`validating.go:23`) |
| 值载体 | `internal/fieldval/fieldval.go`（含已存在但急切装箱的 `NewRV`） |
| 结果输出 | `result.go`（`ValidResult.SafeData/BindStruct` 依赖 `safeData map[string]any`） |
