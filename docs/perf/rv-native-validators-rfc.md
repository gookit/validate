# RFC：validator 入参 RV 化（`internal/validators` 核心 + FieldCtx 自定义校验器）

> **类型**：架构设计 RFC（**待评审，未改业务代码**）。
> **日期**：2026-06-07
> **动机**：架构升级，避免后期扩展受限。v2 起初就想把 validator 入参从 `any` 调整为「封装的 `reflect.Value`」，中途未完成（佐证：`internal/fieldval.FieldValue.NewRV` 注释写着 *"reserved for P2 (struct path will pass the cached reflect.Value here)"*——载体钩子留了，串联没做完）。本 RFC 把这件事补完并定型。
> **前置**：`t3-deboxing-design.md`（去装箱分析）、`checkerr-impl-plan.md`（CheckErr 已落地 = 3 allocs）。

---

## 0. 目标（按优先级）

1. **架构**：把内建校验器收进 `internal/validators`，入参改为**封装的 `reflect.Value`**（carrier），消除「校验器内部反复 `reflect.ValueOf`」的二次反射，建立 RV-native 内核。
2. **扩展性**：自定义校验器**新增**一种参数形态（FieldCtx：直接拿 `reflect.Value`/字段名/args），与现有 `func(val any,...)` **共存**，为后续能力（自定义类型、跨字段、更复杂规则）留口。
3. **兼容**：包级现有 public validator（`Min/IsEmail/...`）与现有 `func(val any,...)` 自定义签名**签名不变、行为不变**。
4. **性能（次要）**：让 `CheckErr` 端到端不装箱 → 逼近 0 alloc，在「只过/败」场景对标 go-playground；**`Check` 路径明确不变**（见 §9 的 safeData 墙）。

> 明确非目标：让 `Check` 变快（safeData 收集强制装箱，RV 化省不掉，见 §9）。

---

## 1. 现状数据流与装箱点（事实基线）

```
applyField (validating.go)
  val,_,_ := v.GetWithDefault(field)        // struct 源在 StructData.TryGet → fv.Interface() 装箱
  fv := fieldval.New(field, val)            // 载体持 any，RV() 懒构造
  callValidator(v, fm, field, val, args, addNum, fv)
     ├─ switch fm.name { case "min": ok = Min(val, args[0]) ... }   // ★ 调 public any 版,Min 内部再 reflect.ValueOf
     └─ default: callValidatorValue(...)    // 自定义/未入 switch → reflect.Call(argIn=[]reflect.Value{...})
```

- 装箱在 **`StructData.TryGet`**（struct 源；map/form 源值本就是 `any`）。
- 内建走 `callValidator` 的 switch，调**public `any` 版**（`Min(val any,...)` → `reflectx.ValueCompare(val,...)` → `reflect.ValueOf(val)` **二次反射**）。
- 自定义走 `callValidatorValue` → `reflect.Call`，已用 `reflect.Value` 组 argIn（值来自 `vfv.RealV()`，无新装箱，但 `val any` 早已装箱）。

载体 `fieldval.FieldValue` 已具备：`RV()`/`RealV()`/`RT()`/`IsZero()`/`IsEmpty()`（全部 lazy+cached），以及**未被使用的 `NewRV(name, rv)`**。

---

## 2. 设计总览（四件事）

1. **`internal/validators`**：内建**无状态**校验器搬入，签名改为 RV-native（吃 carrier / `reflect.Value`）。
2. **public `any` 版变薄 shim**：`validate.Min/IsEmail/...` 保留原签名，内部转 carrier 调 `internal/validators`（仅供**外部直调 / `Val()` / 旧自定义**用）。
3. **`callValidator` 直调 `internal/validators`**（RV-native），**不**经 public shim —— 这才是热路径去二次反射/为去装箱铺路的关键（用户明确点名）。
4. **自定义校验器新增 FieldCtx 形态**：`AddValidator` 额外接受 `func(fl FieldCtx) bool`（typed，直调免 `reflect.Call`），与 `func(val any,...)` 并存（用户明确要求）。

外加（性能可选、可后置）：**端到端 RV 串联**（`tryGetRV` + carrier 持 RV），让 `CheckErr` 真正不装箱。

---

## 3. 载体（carrier）：扩展 `fieldval.FieldValue` + 公开 `FieldCtx`

- 内核继续用 `internal/fieldval.FieldValue`（已 lazy 完备）。把 `NewRV` 正式启用。
- 自定义校验器需要一个**公开**类型 → 在 root 包定义 `FieldCtx` 接口，内部由 `*fieldval.FieldValue` 实现（薄包装，或直接让 root 包一个公开 struct 持有它）。建议**最小接口**：

```go
// FieldCtx 是传给(自定义)校验器的字段级上下文句柄:不装箱地拿到字段值
// (reflect.Value)、字段名/路径与规则参数。命名见 §12 D6(已定 FieldCtx,不跟随
// go-playground 的 FieldLevel——后者源于其 StructLevel/FieldLevel 二分,gookit 无此二分)。
type FieldCtx interface {
    Value() reflect.Value   // 去指针后的字段值(= 内部 RealV)
    Raw() reflect.Value     // 原始 reflect.Value(= 内部 RV)
    FieldName() string      // 字段路径,如 "addr.city"
    Arg(i int) (any, bool)  // 规则第 i 个参数
    Args() []any
    // 可选(跨字段/上下文): Validation() *Validation —— 见 D1,v1 暂不暴露
}
```

> 决策点 D1：`FieldCtx` 是否暴露 `Validation()`（跨字段读其它字段）？go-playground 用 `Top()/Parent()`。建议 v1 先**不**暴露 `*Validation`（避免把内部全摊开），跨字段需求仍走 context validator（§4）；后续按需加。

---

## 4. validator 分类边界（**先划清，决定哪些进 internal**）

| 类别 | 例子 | 形态 | 处置 |
|---|---|---|---|
| **无状态自由函数** | `Min/Max/Lt/Gt/Between/Enum/NotIn/Length/RuneLength/IsInt/IsString/IsNumber/IsEmail/IsURL/IsIP/regexp/isJSON…`（~60+） | 纯函数，不需 `*Validation` | **进 `internal/validators`（RV-native）** |
| **上下文方法** | `Required/RequiredIf/RequiredWith*/EqField/NeField/GtField/LtField…`、文件类 `IsFormFile/IsFormImage/InMimeTypes` | `*Validation` 方法，要读其它字段 / DataFace | **留在 root**（仍 `any`/method；它们读跨字段，依赖 `v`） |

→ `internal/validators` 只收**无状态**那批；上下文校验器不动（它们本就走 `callValidator` 里 `v.Xxx(...)` 分支或 `addNum==2` 的 DataFace 注入）。这条边界是工作量与风险的主控制点。

---

## 5. `internal/validators` 形态 + public shim

内建签名（建议吃 carrier，统一且能复用其 helper；args 已是规则参数）：

```go
// internal/validators
func Min(fl fieldval.FieldValue, min any) bool { /* 用 fl.RV() 比较,不再 reflect.ValueOf */ }
func IsEmail(fl fieldval.FieldValue) bool       { s, ok := fl.String(); return ok && isEmail(s) }
// ... 取串类统一用 fl.String()(从 RV 取,不装箱);标量比较用 fl.RV()/RealV()
```

> 为取串，给 `FieldValue` 加 `String() (string, bool)`（= 现 `valToString` 的 RV 版，命名字符串类型/可转换值都不 panic）。`valToString(any)` 保留为 public 版的薄封装。

public shim（**签名/行为不变**，仅供外部直调/`Val()`/旧自定义）：

```go
// root 包
func Min(val, min any) bool { return validators.Min(fieldval.New("", val), min) }
func IsEmail(val any) bool   { return validators.IsEmail(fieldval.New("", val)) }
```

> 注意 shim 内 `fieldval.New(val)` 仍会在 `RV()` 时 `reflect.ValueOf`——这对**外部直调**无所谓（本来就 any 进来）。热路径不走 shim（见 §6），所以不受影响。

---

## 6. `callValidator` 改造（热路径直调 internal）

把 switch 的每个无状态 case 从「调 public any 版」改为「调 `internal/validators` 的 RV 版，传 carrier」：

```go
// 改前: case "min": ok = Min(val, args[0])
// 改后: case "min": ok = validators.Min(fv, args[0])   // fv 是已构造的 carrier,RV() 复用,无二次反射
```

- 上下文 case（`required`/`eqField`/文件类）**不变**，仍 `v.Xxx(...)`。
- `default`（自定义/未知）→ 见 §7。
- carrier 已在 `valueValidate` 构造（`fv := fieldval.New(field, val)`），直接复用，无新增分配。

**收益（此步单独）**：去掉无状态校验器内部的 `reflect.ValueOf(val)` 二次反射（全路径小幅 CPU），且**不改任何对外签名、不改 alloc**——是**最低风险的子集**，可先单独做（用户若想试水，从这步起）。

---

## 7. 自定义校验器：新增 FieldCtx 形态（additive）

保留旧：`AddValidator("n", func(val any, a1 T1, ...) bool)`（reflect.Call，原样）。
新增：`AddValidator("n", func(fc FieldCtx) bool)`（**typed、直调、免 reflect.Call**）。

实现要点：
1. `checkValidatorFunc`（util.go）识别 func 形态：若入参恰为 `FieldCtx` 接口类型 → 记为 **fieldctx 风格**；否则走旧 `any`/typed 反射风格。
2. `funcMeta` 加判别字段（如 `style uint8`：legacy-reflect / fieldctx）；fieldctx 风格**额外存一个 typed 字段** `fcFunc func(FieldCtx) bool`（避免 reflect.Call 的 argIn 装箱）。
3. `callValidatorValue`（default 分支）按 style 分派：
   - fieldctx：`ok = fm.fcFunc(fcVal)`（直调；args 经 `fc.Arg(i)` 取）。
   - legacy：现有 `reflect.Call` 路径不变。

> 决策点 D2：fieldctx 风格的 args 经 `fc.Arg(i)` 取（运行期 `any`），还是也支持 typed 形参？建议 v1 只支持 `func(FieldCtx) bool`（args 走 `fc.Arg/Args`），最简、免反射；typed-args 留给旧风格。
> 决策点 D3：是否同时给 `AddValidators`/struct 内方法形自定义校验器支持 FieldCtx？建议先只在 `AddValidator`/`v.AddValidator` 支持，struct 方法形后续再说。

---

## 8.（可选/后置）端到端 RV 串联——让 `CheckErr` 不装箱

§6/§7 之后，热路径已是 RV-native，但 `val` 仍在 `TryGet` 处装箱。要让 `CheckErr` 真正省掉装箱：

1. `StructData` 增 `tryGetRV(field) (reflect.Value, exist, zero)`（不 `Interface()`）。
2. 取值链（`tryGet`/`GetWithDefault`/`applyField`）携带 `fieldval.NewRV(field, rv)`（启用那个预留构造），不再 `fieldval.New(any)`。
3. carrier 的 `Src`（`any`）改为**真懒**（只有 legacy 自定义校验器 / safeData 存储 / `fl.Arg` 才触发 `Interface()`）。

**安全边界**：仅 struct 源；map/form 源本就是 `any`，无此装箱。

---

## 9. 预期收益（务必摆正预期）

| 路径 | 现在 | RV 化(§6/§7) | + 端到端(§8) |
|---|---|---|---|
| `Check` / `Struct().Validate()` | 6 / 12 | **不变**（safeData 仍装箱） | **不变** |
| `CheckErr` | 3 | ~3（仅 CPU 改善） | **~0–1**（装箱去掉） |
| ns/op（全路径） | — | 略降（去二次反射） | 略降 |
| 扩展性 / 自定义能力 | any | — | **FieldCtx 自定义校验器可用** |

> **safeData 墙**：`Check` 通过的字段必须装箱进 `safeData map[string]any`，RV 化只是把装箱从 TryGet「挪到」safeData 存储处，净 alloc 不变（详见 `t3-deboxing-design.md` §4）。所以**本 RFC 的 alloc 收益只在 CheckErr**；价值主要是**架构 + 扩展性 + CheckErr 对标 0**。

---

## 10. 分阶段实施（每步独立 commit + golden×3 + race，可单独回滚）

| 步 | 内容 | 风险 | 对外影响 | alloc |
|---|---|---|---|---|
| **R1** | 建 `internal/validators` 脚手架 + 给 `FieldValue` 加 `String()`；**搬 1 个家族**（如比较类 Min/Max/Lt/Gt/Between）到 internal，public 变 shim，`callValidator` 对应 case 直调 internal | 中 | 无（签名不变） | 不变 |
| **R2** | 按家族继续搬（标量类型类 → 取串类 → 其余无状态），每家族一提交 | 中 | 无 | 不变 |
| **R3** | 公开 `FieldCtx` + `checkValidatorFunc` 识别新签名 + `funcMeta.style` + `callValidatorValue` 分派；新增 `func(FieldCtx) bool` 自定义校验器支持 | 中 | **新增 API（附加）** | 不变 |
| **R4**（可选） | `tryGetRV` + carrier 持 RV + `Src` 真懒；measure CheckErr | **高** | 无 | CheckErr ↓ |

> R1~R3 是架构主体（用户的核心诉求：internal 化 + callValidator 直调 + 自定义新参数）。R4 是兑现 CheckErr alloc 的可选续作，风险最高，可独立评估。

---

## 11. 正确性、golden 与回滚

- **golden 是硬证明**：每搬一个家族，`go test -run TestRuleCompat -count=3 .` 必须逐字节不变（校验结果/错误消息不变）。
- **等价性要点**：internal RV 版与 public any 版对**同一输入**必须同判（尤其命名字符串类型、指针、可转换值、nil/零值、`valToString` 的边界）。建议加「public-vs-internal 等价」单测（对一组多样值断言两版结果一致）。
- **carrier 复用**：`callValidator` 复用已构造的 `fv`，注意 `RV()`/`String()` 的 lazy 缓存在一次校验内一致。
- **回滚**：每步独立 commit；R1/R2 任一家族出问题可单独 revert（public shim 与 internal 成对）。R3 新 API 移除不影响旧路径。R4 可整体回退到 §6/§7 状态。

---

## 12. 决策点汇总（评审定）

- **D1** `FieldCtx` 是否暴露 `Validation()`/跨字段访问？（建议 v1 不暴露）
- **D2** 新自定义签名是否只 `func(FieldCtx) bool`（args 走 `fc.Arg`）？（建议是）
- **D3** 新签名支持范围：仅 `AddValidator`，还是也含 struct 方法形自定义器？（建议先仅 `AddValidator`）
- **D4** internal validator 入参用 `fieldval.FieldValue`（值/指针）还是裸 `reflect.Value`+helper？（建议 carrier，统一且复用 lazy）
- **D5** 是否做 R4（端到端去装箱）？还是 R1~R3 架构到位即止、alloc 维持现状？
- **D6** ✅ **已定：`FieldCtx`**（不跟随 go-playground 的 `FieldLevel`——后者是其 StructLevel/FieldLevel 二分的产物，gookit 无此二分；`FieldCtx` 表「字段校验上下文」更贴切，且不与内部 `fieldval.FieldValue` 撞名）。方法：`Value()`/`Raw()`/`FieldName()`/`Arg(i)`/`Args()`。

> **决策状态**：D6 已定。D1–D5 暂取上文「建议默认」（D1 不暴露 `Validation`；D2 仅 `func(FieldCtx) bool` + `Arg`；D3 仅 `AddValidator`；D4 carrier 入参；D5 先做 R1–R3、R4 另议）。**若 `/clear` 后无新指示，按这些默认推进。**

---

## 14. /clear 后如何继续（开发交接）

**当前状态**：本 RFC 已评审定 D6=`FieldCtx`，D1–D5 取建议默认（见 §12）。**尚未写任何业务代码。** 用户已确认按本 RFC 开发。

**第一步 = R1（最低风险，签名/alloc 全不变，先跑通）**：
1. 建 `internal/validators` 包（依赖方向：root → internal/validators → internal/fieldval/reflectx，**勿反向**）。
2. 给 `internal/fieldval.FieldValue` 加 `String() (string, bool)`（= `valToString` 的 RV 版，从 `RV()` 取串，命名字符串类型/可转换值不 panic）。
3. **搬一个家族试水**：比较类 `Min/Max/Lt/Lte/Gt/Gte/Between`（`validators_compare.go`）→ `internal/validators`，签名改吃 `fieldval.FieldValue`（用 `fl.RV()` 比较，去掉内部二次 `reflect.ValueOf`）。
4. root 包 `Min/Max/...` 改为薄 shim：`func Min(val, min any) bool { return validators.Min(fieldval.New("", val), min) }`（签名不变）。
5. `validating.go callValidator` 对应 case 改为直调 `validators.Min(fv, args[0])`（`fv` 是已构造 carrier）。
6. 验收：`go test -run TestRuleCompat -count=3 .`（golden 逐字节）+ `go test -count=1 .` + `go test -race -count=1 .` + `go vet .`；基准 `cd _examples/bench-vs-goplayground && GOWORK=off go test -bench 'Check|CheckErr' -benchmem -run '^$' -benchtime=2s`（alloc 应**不变**：Check 6 / CheckErr 3）。
7. R1 通过后按家族继续 R2，再 R3（FieldCtx 自定义校验器），R4 视 D5。

**环境**：工作目录 `/d/work/inhere/my-tools-dev/gookit2/validate`，master。`go version` 须 go1.25.10，否则 `export GOROOT=/d/work/inhere/linux-env/sdk/gosdk/go1.25.10; export PATH=$GOROOT/bin:$PATH; export GOTOOLCHAIN=local`。

**提交规范**：conventional + scope，body 中文，末尾 `Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>`，不要 push。**每个家族/每步一提交**，golden×3 不留红。

**Git 注意（本仓踩过）**：用户常并行编辑/提交本仓 → `.git/index.lock` 竞争 + 暂存被清。**对已跟踪文件用原子 `git commit <paths> -F -`**；新文件先 `git add` 再在**同一条命令**里 commit；提交后 `git cat-file -p HEAD:<file> | grep` 核验确实落盘。只 add 自己改的文件，**绝不 `git add -A`**；外部并行改动（`.gitignore`、`util.go`、`docs/diff-with-go-playground.md`、`.github/*`、`tmp/` 等）**原样别碰**。

**关键认知（别走偏）**：本重构主收益是**架构 + 扩展性**，不是让 `Check` 变快——`Check` 因 safeData 收集恒装箱（§9 的墙），alloc **不变**；只有 R4 端到端后 `CheckErr` 能降。R1 自检里若看到 Check/CheckErr alloc 没降，是**预期内**。

---

## 13. 代码坐标

| 用途 | 位置 |
|---|---|
| 内建分发 switch | `validating.go callValidator`（~532） |
| 自定义 reflect.Call | `validating.go callValidatorValue` |
| 自定义注册 / 签名校验 | `validation.go AddValidator`(368) / `validators.go AddValidator`(155) / `util.go checkValidatorFunc`(239) |
| funcMeta | `validators.go`（92, `newFuncMeta` 121, `checkArgNum` 105） |
| 载体（待启用 NewRV / 加 String） | `internal/fieldval/fieldval.go` |
| 无状态校验器现位置 | `validators.go` / `validators_compare.go` / `validators_type.go` 等（搬入 `internal/validators`） |
| 上下文校验器（不搬） | `validators.go` 上的 `*Validation` 方法 + `ctxValidatorBuilders`（validate.go:207） |
| 取串 | `validating.go valToString`（23）→ 加 `FieldValue.String()` |
| safeData 墙（为何 Check 不变） | `t3-deboxing-design.md` §4 |
```
