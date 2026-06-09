# gookit/validate vs go-playground/validator 功能差异

> How is `gookit/validate` different from <https://github.com/go-playground/validator>
>
> 对比对象：`gookit/validate` v2 与 `go-playground/validator` v10（基准用 v10.30.3）。
> 性能数字来自 `_examples/bench-vs-goplayground`（同一 i7-14700KF、等价规则、flat 3 字段）。

---

## 0. 一句话定位

- **go-playground/validator**：极致轻量的**纯校验器**（struct + 单值），靠复用单实例、struct 元信息缓存、**只返回 error、不收集结果、不改值**；ns/op 仍是本基准最快（本基准 flat-valid 实测 8 allocs，见 §4）。
- **gookit/validate**：**校验 + 过滤/净化 + 安全数据收集 + 结构体绑定** 一体的「瑞士军刀」，数据源更广（Map/Struct/Request/文件）。多出的几个 alloc 正是这些功能的成本。

---

## 1. 功能差异总表

| 维度 | gookit/validate | go-playground/validator |
|---|---|---|
| 核心定位 | 校验 **+ 过滤 + 收集安全数据 + 绑定** | 专注**校验**（struct + Var 单值） |
| 数据源 | Map / Struct / Request（Form / JSON / url.Values / **文件上传**），按 `Content-Type` 自动收集 | Struct + `Var` / `VarWithValue` |
| 过滤 / 净化 | ✅ 校验前 `filter`（trim / int / upper … 20+），**可改写值** | ❌ 无（只校验、不改值） |
| 结果产出 | ✅ `SafeData()` + `BindStruct` / `BindSafeData`（绑定清洗后数据到结构体） | ❌ 只返回 `error` |
| 成功路径分配 | `Check` 6 / **`CheckErr` 0** allocs（`Check` 因要收集结果） | flat 8 allocs（本基准实测，见 §4） |
| 内建校验器 | **70+**（多数带别名） | 100+（标签关键字） |
| 内建过滤器 | **20+** | — |
| 自定义校验器入参 | `any`（`func(val any, ...) bool`）+ **现已新增 `func(FieldCtx) bool`**（typed、直接拿 `reflect.Value`/字段名/args，与旧签名共存） | `FieldLevel`（直接给 `reflect.Value`） |
| i18n / 错误消息 | **内置** `en` / `zh-CN` / `zh-TW`；struct `message` / `label` tag | 经独立 `universal-translator` 包 |
| 场景（scene） | ✅ 内置场景分组（不同场景校验不同字段） | struct-level + tag / groups（机制不同） |
| 嵌套级联 | **按需**：具名子结构体带 `validate` tag 才下探 | 默认递归；slice/map 元素用 `dive` |
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

1. **更快的成功路径 ns/op**：复用单 `Validate` 实例 + 类型元信息缓存 + 只返回 error，不收集结果；本基准 flat-valid 实测 8 allocs / ~129 B，ns/op 仍领先（~750 vs gookit `CheckErr` ~1155）。
2. **`FieldLevel` 自定义校验器**：自定义校验器拿到的是 `reflect.Value`（`fl.Field()`），无装箱、无二次反射——gookit 现已通过 RV-native 重构补上等价能力（`func(FieldCtx) bool`，见 §5）。
3. **`dive` 深入校验**：对 slice/map 元素、嵌套层级有成熟的 `dive`/`keys`/`endkeys` 语义。
4. **生态规模**：gin 默认绑定校验器，社区、第三方规则、文档更丰富。

---

## 4. 性能对比（flat 3 字段，成功路径）

| 入口 | allocs/op | B/op | ns/op | 说明 |
|---|---:|---:|---:|---|
| `validate.Struct(&u).Validate()` | 7 | ~769 | ~1320 | 通用、非池化；**struct 源只读入口自动跳过 safeData 收集** |
| `validate.Check(&u)` | 6 | ~405 | ~1460 | **推荐默认**，池化 + 收集结果 |
| `validate.CheckErr(&u)` | **0** | **0** | ~1155 | opt-in 快速过/败，不收集结果，**RV-native 端到端去装箱** |
| go-playground `validate.Struct(&u)` | 8 | ~129 | ~750 | 不收集结果，ns/op 仍最快（本基准实测） |

> struct 源只读校验入口（`Validate`/`ValidateErr`/`ValidateE`，只返回 bool/error、不暴露 safeData）现默认走 `skipCollect` 快路径，自动免去 safeData 收集装箱，故 `Struct(&u).Validate()` 从 12 → **7** allocs（写回 `UpdateSource` 开启时跨字段仍正确；map/form 源不跳过）。要取清洗后数据 / 绑定结构体，用 `Check`/`ValidateR`（它们通过 `needCollect` 强制收集）。

> 关键认知：gookit 通过 RV-native 端到端去装箱（载体持 `reflect.Value`、`Src` 懒装箱、`valueCompare`/`required` 纯 RV），`CheckErr` 通过路径已达 **0 alloc**；`Check` 因要收集 `safeData`（供 `SafeData()`/`BindStruct`，每个通过字段都要把值装箱进 `map[string]any`）仍为 6 allocs ——这是「收集结果」功能的固有成本（功能取舍，非实现差）。`Struct().Validate()` 这类只读入口不需要 safeData，已自动跳过收集降到 7 allocs。换言之：本基准 flat-valid 上 gookit `CheckErr` 的 alloc(0) **低于** go-playground(8)，但后者 ns/op 仍更快（~750 vs ~1155）；客观对照、不褒不贬（细节见 `docs/perf/rv-native-validators-rfc.md`）。
>
> gookit 这一轮的演进：flat-valid 成功路径 **23 → 7（非池化 `Struct().Validate()`，struct 源只读自动免收集）/ 6（Check）/ 0（CheckErr）** allocs；完整数字见 `docs/plans/v2.0-perf-bench.txt`。

---

## 5. 架构差异：validator 入参 `any` vs RV-native（`FieldCtx`）

- go-playground 校验器拿 `FieldLevel`，`fl.Field()` 直接是 `reflect.Value`，全程不装箱、不二次反射。
- gookit 旧版内建/自定义校验器入参是 `any`：内建如 `Min(val any,...)` 内部还要 `reflect.ValueOf(val)`（二次反射）；自定义是 `func(val any,...) bool`（装箱 + reflect.Call）。

**RV-native 重构已完成**（不再是架构债）：

1. **内建校验器 internal 化 + 载体持 RV**：~60+ 无状态内建校验器搬入 `internal/validators`，入参为封装的 `reflect.Value`（载体 `internal/fieldval.FieldValue`）；热路径在 `callValidator` 里**直调** internal RV 版，消除「校验器内部反复 `reflect.ValueOf`」的二次反射。public `Min/IsEmail/...` 保留原签名，转为薄 shim（外部直调/`Val()`/旧自定义仍可用），**签名/行为零破坏**。
2. **自定义校验器新增 `func(FieldCtx) bool` 形态**（typed、免 `reflect.Call`，直接拿 `reflect.Value`/字段名/args）。这是**附加形态**，与旧 `func(val any,...) bool` **并存**，旧签名零破坏。命名用 **`FieldCtx`**（本项目选定，非 go-playground 的 `FieldLevel`——后者源于其 StructLevel/FieldLevel 二分，gookit 无此二分；`FieldCtx` 表「字段校验上下文」，且不与内部 `fieldval.FieldValue` 撞名）。
3. **端到端去装箱**：取值链经 `tryGetRV` + `NewRV` 懒构造载体、`Src` 懒装箱、`valueCompare`/`required` 纯 RV，`CheckErr` 通过路径借此达 **0 alloc**（对标 go-playground 的零分配理念）。

> 注意「收集结果」主路径（`Check`）的 6 allocs **不受此影响**：safeData 仍要把通过字段装箱进 `map[string]any`（功能取舍，RV 化省不掉）。`Struct().Validate()` 这类只读入口因不暴露 safeData，已默认跳过收集（12 → 7 allocs）。RV-native 的零分配收益落在 `CheckErr`；主价值是**架构 + 扩展性 + CheckErr 对标 0**。

---

## 6. 选型建议

- 只要「**校验通过/失败**」、追求极致零分配、或已在用 gin 默认校验 → **go-playground**。
- 需要「**过滤净化 + 取清洗后数据/绑定结构体 + 多数据源（Map/请求/文件）+ 内置中文 i18n + 场景**」→ **gookit**。
- 在 gookit 内部按需求选入口：要数据/绑定用 `Check`/`ValidateR`（`*ValidResult`，强制收集 safeData）；只要过/败用 `CheckErr`（struct，最省分配）或 `Struct().Validate()`/`ValidateErr()`（struct 源只读入口默认走 skipCollect 快路径，自动免收集）；map/程序化规则用 `New`/`Map` + `ValidateErr`/`ValidateR`。

---

## 7. 参考

- 对照 benchmark：`_examples/bench-vs-goplayground/`（独立 module，`replace` 指向本地源码）。
- RV-native 重构设计：`docs/perf/rv-native-validators-rfc.md`（本轮 RFC：internal 化 + 载体持 RV + FieldCtx 自定义校验器 + 端到端去装箱）。
- 差距根因分析：`docs/perf/gookit-vs-goplayground-gap-analysis.md`。
- 最新基准：`docs/plans/v2.0-perf-bench.txt`。
