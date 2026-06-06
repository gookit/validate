# gookit/validate vs go-playground/validator 性能差距根因分析与优化清单

> 只读分析文档。结论忠于实测数据，未改动任何业务代码。
> 测量环境：go1.25.10，linux/amd64，Intel i7-14700KF；对照基准位于
> [`_examples/bench-vs-goplayground/`](../../_examples/bench-vs-goplayground)（独立 module，`replace validate => ../..`，go-playground v10.16.0）。
> profiling 原始摘录见仓库 `tmp/perf/`（临时产物，不提交）。

## 一、摘要结论

- **核心差距**：gookit 成功路径慢 ≈4x（1912ns vs 461ns）、多 23 个 alloc，根因是
  **架构差异而非单点 bug**——gookit 每次 `Struct()` 都要**新建一个 `Validation` 实例
  （含 7 个 map + Translator + StructData + 2 个 map）**，并在校验时通过
  **reflect 提取字段值（`fv.Interface()` 装箱）** 和 **reflect 反射调用部分校验器
  （如 `email`）**。这三类成本在每次调用都重复发生。
- **go-playground 为何 0 alloc**：它复用**单个 `Validate` 实例**、把 struct 元信息
  （字段、tag、校验链）**激进缓存进 `sync.Map`**，校验时只读缓存 + 直接 `reflect.Value`
  导航（不调 `.Interface()` 装箱、不走反射函数调用），且**只返回 `error`、成功路径完全
  不写任何结果 map**。成功时无任何堆分配。
- **gookit 的固有成本**：它是"校验 + 过滤 + 绑定"一体化设计——每次校验会收集
  `safeData`（供 `BindStruct`/`SafeData()`）、`filteredData`（供过滤），并为每个实例重建
  规则与翻译器。这部分是设计带来的，**但其中很大一块可通过"复用实例"摊销**。
- **用户的 flag 假设（关 safeData/filter 收集）经实测：收益有限（约 2 alloc / 可忽略 ns）**，
  远不如**复用实例**来得有效。真正的大头是"每次新建实例"的 ~12 个 alloc，已有的
  **Factory（sync.Pool）把 23→11 allocs、1912→1701ns**，这才是与 go-playground
  "复用单实例"对等的正确对照。

---

## 二、Profile 证据

基准（`-benchtime=2s`，每个独立隔离 profile）：

| Benchmark | ns/op | B/op | allocs/op |
|---|--:|--:|--:|
| GookitFlatValid（默认 `Struct()`） | 1912~2013 | 1828~1840 | **23** |
| PlaygroundFlatValid | 461~469 | 0 | **0** |
| GookitFlatValidFactory（复用实例） | **1697~1701** | 920 | **11** |
| GookitNestedValid（默认） | 1854~1935 | 1880 | 24 |
| GookitNestedValidFactory（复用） | **1642** | 977 | **12** |
| PlaygroundNestedValid | 163~166 | 0 | 0 |
| GookitFlatInvalid（失败路径） | 2268~2460 | 3747 | 40 |
| PlaygroundFlatInvalid | 713~772 | 692 | 13 |

### 成功扁平路径 alloc_objects top（隔离 profile `mem_flatvalid.out`）

```
flat   flat%        函数
30.6%  newEmpty (inline)                 — 新建 Validation：7 个 map + struct literal
13.5%  FromStruct                        — 新建 StructData + 2 个 map
10.5%  reflect.packEface                 — fv.Interface() 字段值装箱
10.1%  reflect.Value.call                — email 校验器的反射 Call
 8.9%  (*StructData).instantiateStatic   — 克隆静态规则模板（rules slice + fieldNames）
 8.8%  (*Translator).Reset               — labelMap + fieldMap 两个 map
 4.6%  applyField                        — safeData[field]=val 写入
 4.5%  (*StructData).TryGet              — fieldValues[field]=fv 缓存写入
 3.8%  NewTranslator                     — &Translator{}
```

### CPU top（`cpu_flatvalid.out`）

```
regexp.tryBacktrack / bitState.shouldVisit / MatchRunePos  — email 正则回溯（最大单点 CPU）
reflect.Value.call                                          — email 反射调用
runtime.mapassign_faststr / mapaccess2_faststr             — 7+ 个 map 的写/读
runtime.mallocgcSmallScanNoHeader / scanobject / scavenge   — 分配 + GC 扫描
```

> CPU 上 **email 正则**（`regexp` 回溯）本身就是一大块时间，这部分两个库都跑（go-playground
> 也会做 email 校验），但 gookit 额外叠加了"反射调用 + 23 次 alloc + GC 扫描"的开销。

---

## 三、23 个 allocs 归因表（成功扁平路径，per-op）

> 数据：隔离 profile `mem_flatvalid.out`，alloc_objects 归一化到每次调用（N=1.88M）。
> 分三类：**A=每次新建实例的固定成本**（可被复用消除）、**B=每次解析/装配规则**（可被复用消除）、
> **C=校验执行本身**（复用也无法消除，需改取值/调用路径）。

| # | 分配点（源码位置） | allocs/op | 类别 | 说明 |
|---|---|--:|:--:|---|
| 1 | `newEmpty` `&Validation{}` (validate.go:239) | ~1.5 | A | 实例结构体本身（最大字节块，524MB/总） |
| 2 | `newEmpty` `Errors` map (240) | ~1.4 | A | 错误 map，成功路径根本不用 |
| 3 | `newEmpty` `safeData` map (245) | ~1.3 | A | **safeData 结果 map** |
| 4 | `newEmpty` `optionals` map (246) | ~1.4 | A | 可选字段标记 map |
| 5 | `newEmpty` `validators` map (248) | ~1.6 | A | 实例级校验器名 map |
| 6 | `newEmpty` `validatorMetas` map (250) | ~1.5 | A | 实例级校验器 meta map |
| 7 | `newEmpty` `filteredData` map (252) | ~1.6 | A | **filteredData 结果 map（无 filter 也分配）** |
| 8 | `NewTranslator` `&Translator{}` (messages.go:318) | ~1.3 | A | 翻译器实例 |
| 9 | `Translator.Reset` `labelMap` (329) | ~1.5 | A | 翻译器 labelMap |
| 10 | `Translator.Reset` `fieldMap` (330) | ~1.5 | A | 翻译器 fieldMap |
| 11 | `FromStruct` `&StructData{}` (validate.go:372) | ~1.5 | B | 数据源实例 |
| 12 | `FromStruct` `fieldNames` map (375) | ~1.5 | B | 字段名 map |
| 13 | `FromStruct` `fieldValues` map (376) | ~1.5 | B | 字段值缓存 map |
| 14 | `instantiateStatic` rules slice (struct_rules.go:227) | ~1.5 | B | 每实例独立 rules slice（避免共享模板被改） |
| 15 | `instantiateStatic` `fieldNames[k]` 写入 (261) | ~1.5 | B | 复制字段名（map grow） |
| 16 | `TryGet` `fieldValues[field]=fv` (data_source.go:457) | ~1.5 | C | 字段值缓存写入（map grow） |
| 17 | `TryGet` `fv.Interface()` packEface (345/463) | ~3.5 | **C** | **3 个字段值的 reflect 装箱（每字段一次）** |
| 18 | `applyField` `safeData[field]=val` (validating.go:214) | ~1.5 | C | safeData 写入（含装箱后的值落 map） |
| 19 | `callValidatorValue` `fv.Call(argIn)` (741) | ~3.4 | **C** | **email 走反射 Call：argIn slice + 结果装箱** |

> 合计约 23 allocs。类别 A≈12、B≈7.5、C≈9（采样有重叠/抖动，故略大于 23；
> 关键是**量级与归属**而非小数点）。

**两条最值得关注的"执行类"分配（C）**：

- **#17 `fv.Interface()`（packEface）≈3.5 alloc**：每个字段值从 `reflect.Value` 取出为
  `any` 都要装箱。go-playground 不这么做——它直接用 `reflect.Value` 做比较/长度，
  避免装箱。
- **#19 email 反射 Call ≈3.4 alloc**：`email→isEmail` 注册为全局 `reflect.ValueOf(IsEmail)`
  （register.go:79），**不在 `callValidator` 的 switch 里**，故落到 `default` 分支走
  `fv.Call(argIn)`：要 `make([]reflect.Value, ...)` + 结果 `.Bool()` 装箱。
  而 `required/min/max/min_len/max_len` 都在 switch 里，零反射。

---

## 四、go-playground 为什么 0 分配（设计对比）

| 维度 | go-playground/validator | gookit/validate v2 |
|---|---|---|
| 实例 | 全局复用**单个** `*Validate` | **每次** `Struct()` 新建 `Validation` |
| struct 元信息 | 首次解析后存入 `sync.Map`（`structCache`/`tagCache`），后续纯读 | 有类型 meta 缓存 + 静态模板，但**仍每实例 clone 规则/字段名/翻译表** |
| 取值 | 直接持 `reflect.Value`，比较/取长度**不装箱** | `TryGet` 调 `fv.Interface()` 把每个字段**装箱为 `any`** |
| 校验器调用 | 内置校验器是 `map[string]Func`，**直接函数调用**，无反射 | 部分内置走 switch（零反射），**email/url 等仍走 `fv.Call` 反射** |
| 结果收集 | 成功**只返回 `nil error`**，不写任何 map | 成功要写 `safeData[field]`（供 BindStruct）+（有 filter 时）`filteredData` |
| 错误对象 | 失败时才 `make` 错误切片 | 实例创建时就 `make(Errors)`（成功也分配） |

**一句话**：go-playground 把"一次性成本"（解析 struct）缓存掉，把"每次成本"压到
"读缓存 + 无装箱 reflect 导航 + 直接函数调用 + 不收集结果"，于是成功路径 0 alloc。
gookit 的每次新建实例 + 装箱取值 + 反射调用 + 结果收集，构成了这 23 个 alloc。

---

## 五、用户 flag 假设的实测验证（关 safeData / filter 收集）

用户假设：加 opt-in flag 跳过 `safeData`/`filteredData` 收集来降开销。**实测结论：收益很小。**

实验方法：临时改源码加 flag（测后已全部 revert，业务代码零改动），用
`testing.AllocsPerRun` + 隔离 benchmark 量化。

| 实验 | allocs/op | vs 基线 |
|---|--:|--:|
| A 基线（默认 `Struct()`） | 23 | — |
| B 仅跳过每字段 `safeData[field]=val` 写入（保留 map） | 27 | **+4（更差）** |
| C 仅跳过 `safeData`+`filteredData` 两个 `make()`（保留写入） | 25 | **+2（更差）** |
| D 同时跳过写入 + 两个 map | 25 | **+2（更差）** |

**为什么"省不到反而更多"**：

1. **两个结果 map 的 `make()` 各约 1 alloc（合计 2 alloc / 每个约 48–55B）**，这是
   `safeData`/`filteredData` 的全部"map 分配"成本——**确实存在但很小**。
2. **每字段 `safeData[field]=val` 写入本身≈不额外分配**：map 初始 bucket 够装 3 个键，
   写入命中已分配的 bucket，不触发 grow。
3. 把 map 设为 `nil` 后，字段值 `any`（packEface 装箱结果）**失去了"被长期 map 引用"
   的逃逸去向**，编译器逃逸分析改变，反而在 `fv.Interface()` 处多出装箱——这正是 B/C/D
   反而 +2~+4 的原因。**说明真正的分配大头是 `fv.Interface()` 装箱（#17），而非 safeData
   这个 map。**

**filter 链路开销**：

- `filteredData` map **无论有没有 filter 规则都会在 `newEmpty()` 里无条件 `make()`**
  （1 alloc/op），属"白交的税"。
- 但 `Filtering()` 在无 filter 规则时只是 `for range v.filterRules`（空切片）立即返回，
  **不写 `filteredData`、无额外分配**。
- 所以 filter 链路的成本 = **1 个 `filteredData` 空 map 的 alloc**（无规则时）。

**结论**：opt-in flag 跳过 safeData/filteredData 收集，理论上限省 **2 个 map alloc / ≈0 ns**，
且实现要绕开逃逸分析陷阱（直接不 `make` 才有效，置 nil 无效）。**值得做但不是大头**，
不应作为首要优化。

---

## 六、Factory 复用实测（与 go-playground 对等对照）

仓库已有 `factory.go`（`NewFactory` + `sync.Pool` + `resetForReuse`）。补一个复用 benchmark
做"复用实例"的对等对照（go-playground 也是复用单实例）：

| 路径 | ns/op | allocs/op | B/op |
|---|--:|--:|--:|
| 默认 `Struct()`（每次新建） | 1912 | 23 | 1828 |
| **Factory 复用（pool + 缓存 meta）** | **1701** | **11** | **920** |
| 收益 | **−11%（−211ns）** | **−12 allocs（−52%）** | **−908B** |

嵌套场景：24→**12** allocs、1935→**1642** ns。

**解读**：

- 复用消除了**全部"类别 A"分配**（newEmpty 的 7 个 map + Translator 的 3 个 alloc，
  共 ~12 个），从 `mem_factory.out` 看，pool 路径剩下的 11 个 alloc 全部来自：
  `FromStruct`（新建 StructData + 2 map，**未被 pool 复用**）、`instantiateStatic`
  （clone rules/fieldNames）、`TryGet`（`fieldValues` 写 + 3× `fv.Interface` 装箱）、
  `reflect.Value.call`（email 反射）、`safeData` 写入。
- 即"复用实例"砍掉了一半 alloc，但 **ns 只降 11%**——因为剩下的时间大头是
  **email 正则 + 反射调用 + StructData 重建 + 装箱取值**，这些复用 `Validation` 实例
  并不能消除（StructData 每次仍 `FromStruct` 新建）。
- 要逼近 go-playground，还需：**缓存/复用 StructData 与 fieldValues**、**避免 `fv.Interface()`
  装箱**、**把 email 等校验器搬进 switch 免反射**。

---

## 七、优化机会清单（按收益/成本排序）

> 收益数字基于上文实测；"是否默认可改"指能否在不破坏 golden 输出/向后兼容的前提下默认开启。

### T1. 把 Factory 复用做成"推荐用法"+ 把 StructData/fieldValues 也纳入复用

- **是什么**：Factory 已能复用 `Validation`（−12 alloc）。进一步让 `FromStruct` 的
  `StructData`（含 `fieldNames`/`fieldValues` 两个 map）也走 pool 或随类型 meta 缓存，
  消除剩余的 #11–#13、#16。
- **预计收益**：当前 Factory 已实测 **23→11 allocs、−211ns**；再复用 StructData 预计可
  再去掉 ~3–4 alloc（FromStruct 的 struct + 2 map）。
- **风险/复杂度**：中。`fieldValues` 缓存的是**具体值的 reflect.Value**，跨次校验不同实例
  必须清空，`resetForReuse` 已有先例可循；StructData 入池需保证 reset 干净。
- **opt-in / 默认**：**opt-in**（Factory 本就是 opt-in）。不影响默认 `Struct()` 语义。
- **golden/兼容**：不影响（结果一致）。
- **建议**：**最高优先**——这是唯一已被实测验证、收益最大且无兼容风险的方向。
  当前 Factory 文档可见度低，建议在 README/文档显著推荐"批量校验用 Factory"。

### T2. 把常用字符串校验器搬进 `callValidator` 的 switch（免反射 Call）✅ 已实施

- **是什么**：`email`(`isEmail`)、`url`、`ip`、`alpha` 等当前注册为全局
  `reflect.ValueOf(fn)`，走 `default→fv.Call` 反射分支。把它们像 `min/max/isInt` 一样
  加进 switch，直接函数调用。
- **预计收益**：实测 email 反射 Call ≈ **3.4 alloc/op + 显著 CPU**（`reflect.Value.call`
  占 10% CPU）。每个改造的校验器在用到时省 ~3 alloc + 反射开销。
- **风险/复杂度**：低。纯加 case，逻辑等价；需保证签名/参数与原反射调用一致。
- **opt-in / 默认**：**默认可改**（行为完全等价）。
- **golden/兼容**：不影响（同一个 `IsEmail`，只是调用方式不同）。
- **建议**：**次高优先**，低风险高收益，对所有用 email/url 等校验的成功**和**失败路径都有效。

**✅ 已实施（实测）**：在 `validating.go` 的 `callValidator` switch 新增 **31 个**单参数自由函数
校验器 case，全部直接函数调用、免反射 `fv.Call`：

- 接收 `any` 的直传 `val`：`isNumeric`（`isNumber` 此前已在 switch）。
- 接收 `string` 的统一走 `valToString(val)` 安全取串（与现有 `regexp`/`isJSON`/`isStringNumber`
  写法一致，命名字符串类型/可转换值都不 panic，取不到串时保持失败，行为与反射路径逐字一致）：
  `isEmail`、`isURL`、`isFullURL`、`isIP`、`isIPv4`、`isIPv6`、`isCIDR`、`isCIDRv4`、`isCIDRv6`、
  `isMAC`、`isAlpha`、`isAlphaNum`、`isAlphaDash`、`isASCII`、`isPrintableASCII`、`isUUID`、
  `isUUID3`、`isUUID4`、`isUUID5`、`isBase64`、`isDataURI`、`isHexadecimal`、`isHexColor`、
  `isRGBColor`、`isLatitude`、`isLongitude`、`isDNSName`、`isMultiByte`、`isCnMobile`、
  `isISBN10`、`isISBN13`、`hasWhitespace`。
- 未纳入：`isActiveURL`（做真实网络 HTTP 请求，非纯校验，热路径不适用）。

实测对照（`_examples/bench-vs-goplayground`，go1.25.10 / i7-14700KF，`-benchtime=2s`，含 email 字段）：

| 场景 | 改前 ns/op | 改后 ns/op | 改前 allocs | 改后 allocs | 改前 B/op | 改后 B/op |
|---|--:|--:|--:|--:|--:|--:|
| GookitFlatValid（成功，含 email） | 2020 | **1853** | 23 | **21** | 1842 | 1808 |
| GookitFlatInvalid（失败，email 非法） | 2485 | 2464 | 40 | 40 | 3748 | 3749 |

> 成功路径 email 校验从反射 `fv.Call` 改直调，**−2 allocs（23→21）/ −167ns（≈−8%）**，与
> §三归因表 #19（email 反射 Call ≈3.4 alloc）吻合（采样抖动 + step3 已把值转为 string 故落到
> ~2）。失败路径 40 allocs 不变：email 同样改直调省下的分配被失败分支的错误消息构建主导，
> 此基准看不到差异（但反射 CPU 仍被消除）。golden（`TestRuleCompat`×3）逐字节不变。

### T3. 避免 `TryGet` 的 `fv.Interface()` 装箱（取值路径）

- **是什么**：`TryGet` 把每个字段 `fv.Interface()` 装箱成 `any` 返回（#17 ≈3.5 alloc）。
  对内置校验器（走 switch 的），可考虑沿用已有的 `fieldval.FieldValue` 载体（valueValidate
  里已构造 `fv := fieldval.New(...)` 复用 `rV()`），把"原始 reflect.Value"一路带到校验函数，
  对数值/字符串比较直接用 `reflect.Value` 而非 `any`。
- **预计收益**：理论可省到 ~3 alloc/op（每字段一次装箱）；是除实例复用外**最大的单点
  执行类分配**。
- **风险/复杂度**：**高**。会触及 `Get/GetWithDefault/tryGet/callValidator` 整条取值与
  分发链，且大量内置/自定义校验器签名是 `func(val any, ...)`，要么改签名要么做双路径。
  容易引入回归。
- **opt-in / 默认**：建议先 opt-in 或仅对内置 switch 校验器内部优化，不改公开签名。
- **golden/兼容**：高风险，需大量回归测试保证。
- **建议**：**中优先、需专项设计**。收益大但改动面广，应单独立项 + 充分 golden 覆盖。

### T4. `filteredData` 改为按需（懒）分配 ✅ 已实施（与 T6 合并为 Step 2 专项）

- **是什么**：`newEmpty()` 无条件 `make(filteredData)`。无 filter 规则时这是白交的税。
  改为首次写 filter 结果时才 `make`（lazy）。
- **预计收益**：无 filter 时省 **1 alloc/op**（实测 map ≈48–55B）。
- **风险/复杂度**：低，但要审计所有读 `filteredData`/`Filtered()` 的地方对 nil map 安全
  （Go 读 nil map 安全，写需先建）。
- **opt-in / 默认**：**默认可改**（懒分配对外部不可见）。
- **golden/兼容**：不影响。
- **建议**：**低优先、顺手做**。单独收益小（1 alloc），可与 T5 合并。

### T5. `safeData` / `Errors` 结果 map 懒分配 ✅（懒分配部分已实施）+ opt-in 关收集（未做）

- **是什么**：(a) `safeData`、`Errors` 同样可懒分配（成功路径不碰 Errors，无需 BindStruct
  时不碰 safeData）。(b) 提供 opt-in flag（如 `v.CollectSafeData=false`）让不需要
  `BindStruct/SafeData()` 的调用方完全跳过 safeData 收集。
- **预计收益**：实测**上限 ≈2 alloc/op（safeData+filteredData 两个 map），ns 基本不变**。
  注意：必须是"不 `make`"才有效，置 nil 反而因逃逸分析变差（见 §五）。
- **风险/复杂度**：中。懒分配要审计 `tryGet`（从 safeData 读缓存）、`Validate` 错误路径
  `v.safeData = make(...)`、`BindSafeData` 等所有触点；opt-in flag 要在 `resetForReuse`
  里正确重置。
- **opt-in / 默认**：懒分配**默认可改**；关收集是 **opt-in**。
- **golden/兼容**：懒分配不影响；关收集后 `SafeData()`/`BindStruct` 返回空——**会改变
  这两个 API 的行为**，必须 opt-in 且文档说明。
- **建议**：**低优先**。这正是用户假设的方向，**实测确认收益有限（≤2 alloc）**，不应
  作为首要项；可作为 T1/T4 的附带收尾。

### T6. `Translator` 两个 map（labelMap/fieldMap）懒分配 ✅ 已实施（Step 2 专项）

- **是什么**：`NewTranslator→Reset` 无条件建 `labelMap`+`fieldMap`（#9/#10 ≈3 alloc）。
  绝大多数校验没有自定义 label/field 映射，可懒分配。
- **预计收益**：~2 alloc/op（无自定义翻译时）。
- **风险/复杂度**：低–中，需审计 `FieldName`/`AddLabelMap`/`addLabelName` 对 nil map 的读写。
- **opt-in / 默认**：**默认可改**。
- **golden/兼容**：不影响。
- **建议**：低优先，可与 T4/T5 一起做"newEmpty 全面懒分配"专项。

---

## Step 2 — `newEmpty()` / `Translator` 全面懒分配（T4 + T6 + T5 懒分配部分）✅ 已实施

**做了什么**：把 `newEmpty()` 里原本无条件 `make()` 的 6 个实例 map（`Errors`、`safeData`、
`filteredData`、`optionals`、`validators`、`validatorMetas`）与 `NewTranslator()→Reset()` 里的
`labelMap`/`fieldMap`，全部改为 **首次写入时才 `make()`（懒分配）**。读 nil map 安全（返回零值），
仅写入需守卫，故每个写入点前加 `ensure*()` 守卫（`ensureErrors/ensureSafeData/ensureFilteredData/
ensureOptionals/ensureValidatorMaps`，定义在 `validation.go`；translator 两 map 内联守卫于
`addLabelName`/`AddFieldMap`）。

- 写入点覆盖（守卫数）：`Errors` 1 处（`AddError`）；`safeData` 3 处（filtering.go×1 + validating.go×2）；
  `filteredData` 3 处（filtering.go×2 + validating.go×1）；`optionals` 2 处入口
  （rule.go 解析 + struct_rules.go 模板回放；`isInOptional` 内 3 处只改写**已存在**的 key，
  在 `range v.optionals` 内执行，nil 时不进入循环，无需守卫）；`validators`+`validatorMetas`
  3 处（`AddValidator` + `validatorMeta` 的 ctx/struct 两分支，合用 `ensureValidatorMaps`）；
  `labelMap`/`fieldMap` 各 1 处（`addLabelName`/`AddFieldMap`，含空入参短路）。
- `ResetResult()`/`resetRules()` 改为把结果/optionals map 置 `nil`（而非 `make`），让 `Reset()`
  后的实例在下一次干净校验仍保持零分配；`resetForReuse()`（Factory）用 `clear(v.xxx)`，
  `clear(nil)` 在 Go1.21+ 是安全 no-op，复用语义不变（首次 newEmpty 得 nil → 写入时懒建 →
  reset 后再用仍走懒建）。
- 全部保持对外行为不变（golden `TestRuleCompat`×3 逐字节不变）。

**实测对照**（`_examples/bench-vs-goplayground`，go1.25.10 / i7-14700KF，`-benchtime=2s -run '^$'`）：

| 场景 | 改前 ns/op | 改后 ns/op | 改前 allocs | 改后 allocs | 改前 B/op | 改后 B/op |
|---|--:|--:|--:|--:|--:|--:|
| GookitFlatValid（成功扁平） | 1883 | **1687** | 21 | **14** | 1808 | 1465 |
| GookitFlatInvalid（失败扁平） | 2501 | **2332** | 40 | **34** | 3748 | 3460 |
| GookitNestedValid（成功嵌套） | 1954 | **1775** | 24 | **17** | 1880 | 1544 |

> 成功路径 **−7 allocs（21→14）/ −196ns（≈−10%）**；失败路径 **−6 allocs（40→34）/ −169ns**；
> 嵌套成功 **−7 allocs（24→17）/ −179ns**。超出 T4/T6 合计 ~5–6 alloc 的预估（实测 7），
> 因成功路径同时省下 Errors、validators、validatorMetas、optionals、labelMap、fieldMap 共 6 个
> 从不写入的 map，外加 filteredData（无 filter）。验证全绿：golden×3 / 全量 / **race** / vet /
> Factory 测试（含并发）。

---

## 八、建议落地顺序

1. **T2（switch 收纳 email/url 等）** — 低风险、默认可改、对成功+失败路径都有效，先做。
2. **T1（推广 Factory + StructData 入池）** — 已验证收益最大（−12 alloc/−211ns），
   文档先推荐用法，再做 StructData 复用增量。
3. **T4 + T6 + T5(懒分配部分)** 合并为"**`newEmpty`/`Translator` 全面懒分配**"专项 —
   **✅ 已实施（见上 Step 2）**：统一把 Errors/safeData/filteredData/optionals/validators/
   validatorMetas/labelMap/fieldMap 改懒分配，成功路径实测 **−7 alloc（21→14）/ −196ns（≈−10%）**。
4. **T5(opt-in 关收集)** — 作为上一项的 opt-in 收尾，收益小，按需提供 API。
5. **T3（去装箱取值路径）** — 单独立项 + 充分 golden 覆盖，收益大但改动面广，最后做。

> **总体判断**：用户"关 safeData/filter 收集"的方向**实测收益有限（≤2 alloc，ns 基本不变）**，
> 不是性价比最高的优化。**最有效的方向是"复用实例（Factory，已落地，23→11/−211ns）"
> 与"消除 reflect 装箱/反射调用（T2/T3）"**。建议优先推广 Factory + 做 T2。
