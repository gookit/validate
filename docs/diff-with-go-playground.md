# gookit/validate vs go-playground/validator 功能差异

> How is `gookit/validate` different from <https://github.com/go-playground/validator>
>
> 对比对象：`gookit/validate` v2 与 `go-playground/validator` v10（基准用 v10.30.3）。
> 性能数字来自 `_examples/bench-vs-goplayground`（同一 i7-14700KF、等价规则、flat 3 字段）。

---

## 0. 一句话定位

- **go-playground/validator**：极致轻量的**纯校验器**（struct + 单值），成功路径 **0 alloc**——靠复用单实例、struct 元信息缓存、**只返回 error、不收集结果、不改值**。
- **gookit/validate**：**校验 + 过滤/净化 + 安全数据收集 + 结构体绑定** 一体的「瑞士军刀」，数据源更广（Map/Struct/Request/文件）。多出的几个 alloc 正是这些功能的成本。

---

## 1. 功能差异总表

| 维度 | gookit/validate | go-playground/validator |
|---|---|---|
| 核心定位 | 校验 **+ 过滤 + 收集安全数据 + 绑定** | 专注**校验**（struct + Var 单值） |
| 数据源 | Map / Struct / Request（Form / JSON / url.Values / **文件上传**），按 `Content-Type` 自动收集 | Struct + `Var` / `VarWithValue` |
| 过滤 / 净化 | ✅ 校验前 `filter`（trim / int / upper … 20+），**可改写值** | ❌ 无（只校验、不改值） |
| 结果产出 | ✅ `SafeData()` + `BindStruct` / `BindSafeData`（绑定清洗后数据到结构体） | ❌ 只返回 `error` |
| 成功路径分配 | `Check` 6 / `CheckErr` 3 allocs（**因要收集结果**） | **0 alloc** |
| 内建校验器 | **70+**（多数带别名） | 100+（标签关键字） |
| 内建过滤器 | **20+** | — |
| 自定义校验器入参 | `any`（`func(val any, ...) bool`） | `FieldLevel`（直接给 `reflect.Value`） |
| i18n / 错误消息 | **内置** `en` / `zh-CN` / `zh-TW`；struct `message` / `label` tag | 经独立 `universal-translator` 包 |
| 场景（scene） | ✅ 内置场景分组（不同场景校验不同字段） | struct-level + tag / groups（机制不同） |
| 嵌套级联 | **按需**：具名子结构体带 `validate` tag 才下探（Java `@Valid` 风格） | 默认递归；slice/map 元素用 `dive` |
| slice/map 元素 | `field.*` 通配（`v.StringRule("tags.*", "...")`） | `dive` 关键字 |
| 规则语法 | `\|` 分隔、`:` 传参（`required\|min:1\|max:99`） | `,` 分隔、`=` 传参（`required,min=1,max=99`） |
| 跨字段 | `eq_field` / `gt_field` / `required_with` … | `eqfield` / `gtfield` / `required_with` … |
| 单值校验 | `validate.Val("x@a.com", "required\|email")` | `validate.Var(x, "required,email")` |
| 生态 / 成熟度 | 中文生态友好、功能全面 | 更广（gin 默认）、社区 / star 更大 |

---

## 2. gookit 独有 / 更强

1. **过滤与净化（filter）**：校验**之前**对值做转换/清洗（`trim`、`int`、`lower`、`title`…），并写回源数据。go-playground 不改值。
2. **安全数据收集 + 绑定**：`Check(&u)` / `v.ValidateR()` 返回 `*ValidResult`，可 `SafeData()` 取「只含被规则校验过的字段」的干净数据，或 `BindStruct(ptr)` 直接绑定。
3. **多数据源**：原生支持 `Map`、`http.Request`（按 `Content-Type` 自动收集 Form/JSON/Query）、**文件上传**校验（`isFile`/`isImage`/`inMimeTypes`）。
4. **场景（scene）**：一套规则按场景分组，不同场景只校验对应字段。
5. **内置 i18n**：`en`/`zh-CN`/`zh-TW` 开箱即用；结构体 `message`/`label` tag 直接定制消息与字段翻译。
6. **别名丰富**：多数校验器/过滤器有别名（`min_len`/`minLen`/`minlen`…）。

## 3. go-playground 独有 / 更强

1. **0 alloc 成功路径**：复用单 `Validate` 实例 + 类型元信息缓存 + 只返回 error，不收集结果、不装箱。
2. **`FieldLevel` 自定义校验器**：自定义校验器拿到的是 `reflect.Value`（`fl.Field()`），无装箱、无二次反射——这也是 gookit 的一处架构差（见 §5）。
3. **`dive` 深入校验**：对 slice/map 元素、嵌套层级有成熟的 `dive`/`keys`/`endkeys` 语义。
4. **生态规模**：gin 默认绑定校验器，社区、第三方规则、文档更丰富。

---

## 4. 性能对比（flat 3 字段，成功路径）

| 入口 | allocs/op | B/op | ns/op | 说明 |
|---|---:|---:|---:|---|
| `validate.Struct(&u).Validate()` | 12 | ~1101 | ~1519 | 通用、非池化 |
| `validate.Check(&u)` | 6 | 405 | ~1390 | **推荐默认**，池化 + 收集结果 |
| `validate.CheckErr(&u)` | 3 | 40 | ~1147 | opt-in 快速过/败，不收集结果 |
| go-playground `validate.Struct(&u)` | **0** | 0 | ~470–750 | 不收集结果 |

> 关键认知：**与 0 的差距本质是功能取舍**，不是单纯的实现差。gookit 收集 `safeData`（供 `SafeData()`/`BindStruct`），每个通过字段都要把值装箱进 `map[string]any` ——这是「收集结果」功能的固有成本。`CheckErr` 通过「不收集结果 + 同字段装箱去重」逼近到 3 allocs，是这条线上能做到的实用下限（细节见 `docs/perf/t3-deboxing-design.md` §4、`docs/perf/checkerr-impl-plan.md`）。
>
> gookit 这一轮的演进：flat-valid 成功路径 **23 → 12（非池化）/ 6（Check）/ 3（CheckErr）** allocs；完整数字见 `docs/plans/v2.0-perf-bench.txt`。

---

## 5. 架构差异：validator 入参 `any` vs `FieldLevel`

- gookit 内建/自定义校验器入参是 `any`：内建如 `Min(val any,...)` 内部还要 `reflect.ValueOf(val)`（二次反射）；自定义是 `func(val any,...) bool`（装箱 + reflect.Call）。
- go-playground 校验器拿 `FieldLevel`，`fl.Field()` 直接是 `reflect.Value`，全程不装箱、不二次反射。
- gookit 内部其实已有载体雏形 `internal/fieldval.FieldValue`（带 lazy `RV()`），但它**不是**公开的校验器入参契约。

这是一笔**已知、有意保留的架构债**：在「收集结果」主路径上把它还掉收益 ≈ 0（safeData 仍要装箱）；要还需重写 ~70 个内建校验器并**改动公开的自定义校验器签名**（破坏所有用户的自定义 validator）。它是让 `CheckErr` 进一步逼近 0 alloc 的钥匙，但属大破坏性变更，目前不动；若推进，建议以**新增** FieldLevel 风格的可选接口、与现有 `any` 版共存的方式渐进。

---

## 6. 选型建议

- 只要「**校验通过/失败**」、追求极致零分配、或已在用 gin 默认校验 → **go-playground**。
- 需要「**过滤净化 + 取清洗后数据/绑定结构体 + 多数据源（Map/请求/文件）+ 内置中文 i18n + 场景**」→ **gookit**。
- 在 gookit 内部按需求选入口：要数据/绑定用 `Check`/`ValidateR`（`*ValidResult`）；只要过/败用 `CheckErr`（struct，最省分配）；map/程序化规则用 `New`/`Map` + `ValidateErr`/`ValidateR`。

---

## 7. 参考

- 对照 benchmark：`_examples/bench-vs-goplayground/`（独立 module，`replace` 指向本地源码）。
- 性能演进与设计：`docs/perf/v2.x-perf-optimization-plan.md`、`docs/perf/t3-deboxing-design.md`、`docs/perf/checkerr-impl-plan.md`。
- 最新基准：`docs/plans/v2.0-perf-bench.txt`。
