# gookit/validate vs go-playground/validator 对照 benchmark

对照 [gookit/validate](https://github.com/gookit/validate) v2 与
[go-playground/validator](https://github.com/go-playground/validator) v10 在
**等价规则下的校验调用** 性能。对应 issue
[#215](https://github.com/gookit/validate/issues/215)（请求补一份 vs go-playground 的 benchmark）。

## 这是一个独立 module，不影响主仓库依赖

本目录有 **自己独立的 `go.mod`**（`module github.com/gookit/validate/v2/_examples/bench-vs-goplayground`），
通过 `replace` 指向上层本地 validate 源码：

```
require (
    github.com/go-playground/validator/v10 v10.16.0
    github.com/gookit/validate/v2 v2.0.0
)

replace github.com/gookit/validate/v2 => ../..
```

因此：

- `go-playground/validator` 及其传递依赖 **只属于这个子 module**，不会进入主仓库的
  `go.mod` / `go.sum`。
- `_examples/` 被 Go 工具默认忽略，主仓库根目录 `go test ./...` 不会进入此目录。
- 主仓库 `go.mod` / `go.sum` 保持零改动。

## 如何运行

```bash
cd _examples/bench-vs-goplayground
go test -bench . -benchmem -run '^$'
```

- `-run '^$'` 跳过单测（本目录无单测，仅 benchmark），只跑基准。
- `-benchmem` 输出 `B/op` / `allocs/op`。
- 如本机存在根目录 `go.work`，可加 `GOWORK=off` 让子 module 独立解析依赖：
  `GOWORK=off go test -bench . -benchmem -run '^$'`。
- 首次运行需联网拉取 go-playground 及其传递依赖（`go mod tidy` 已固定版本）。

## 对照场景与 Benchmark 函数

对 **同一组等价场景** 分别用两库各跑一个 benchmark，字段数与规则语义尽量一致。
规则语法差异：gookit 用 `|` 分隔、`:` 传参；go-playground 用 `,` 分隔、`=` 传参。

| 场景 | gookit 函数 | go-playground 函数 | 规则（语义等价） |
|------|-------------|--------------------|------------------|
| 扁平 struct · 成功路径 | `BenchmarkGookitFlatValid` | `BenchmarkPlaygroundFlatValid` | `required` + 长度 3~20 + email + 数值 1~120 |
| 扁平 struct · 失败路径 | `BenchmarkGookitFlatInvalid` | `BenchmarkPlaygroundFlatInvalid` | 同上，但数据非法（名字过短 / email 非法 / 越界） |
| 嵌套 struct · 成功路径 | `BenchmarkGookitNestedValid` | `BenchmarkPlaygroundNestedValid` | 外层 + 一层嵌套子 struct 字段 |

规则对应关系：

| 语义 | gookit | go-playground |
|------|--------|---------------|
| 必填 | `required` | `required` |
| 字符串长度 3~20 | `min_len:3\|max_len:20` | `min=3,max=20` |
| email | `email` | `email` |
| 数值 1~120 | `min:1\|max:120` | `gte=1,lte=120` |

## 公平性声明（重要）

两库定位与语义并不相同，对照仅就 **“等价规则的校验调用”** 而言：

- **特性范围不同**：gookit/validate 是 **校验 + 过滤(filter) + 数据绑定(bind)** 一体；
  go-playground/validator **专注校验**。功能集不同，单看 ns/op 不代表全部价值。
- **调用模型不同**：gookit 当前 public API 每次 `validate.Struct(&data)` 会构建一个
  `Validation` 实例并解析 struct tag（内部有 typeMeta 缓存，但实例本身每轮新建）；
  go-playground 复用一个 `validator.New()` 实例、结构体元信息进内部缓存，循环内只做校验。
  两边都遵循“循环外建好可复用的东西、循环内只跑校验”，但 gookit 每轮的实例构建是其
  真实使用成本，如实计入。
- **级联语义不同**（嵌套场景）：
  - gookit：子 struct 字段需带 `validate` tag（空 tag `validate:""` 也算）才会级联校验子字段，
    这里给 `Profile` 加 `validate:"required"` 触发级联；
  - go-playground：嵌套 struct 默认递归校验其字段，无需特殊 tag；`dive` 用于 slice/map
    元素（本例是单个嵌套 struct，故不需要 `dive`）。
- **规则语法差异**：见上表，`|`/`:` vs `,`/`=`。

结论：本基准衡量的是“相同校验语义下两库各自路径的开销”，**不是** 库的整体优劣判断。

## 本地运行结果（如实记录，勿夸大）

环境：

| 项 | 值 |
|----|----|
| goos / goarch | linux / amd64 |
| cpu | Intel(R) Core(TM) i7-14700KF |
| Go | go1.25.10 |
| 并行度 | GOMAXPROCS=28（基准后缀 `-28`） |
| 取值 | 两次运行结果一致（噪声内），下表取其一 |

```
BenchmarkGookitFlatValid-28          589628    2016 ns/op    1844 B/op    23 allocs/op
BenchmarkPlaygroundFlatValid-28     2558196    470 ns/op        0 B/op     0 allocs/op
BenchmarkGookitFlatInvalid-28        480970    2468 ns/op    3751 B/op    40 allocs/op
BenchmarkPlaygroundFlatInvalid-28   1603652    747 ns/op      692 B/op    13 allocs/op
BenchmarkGookitNestedValid-28        606098    1959 ns/op    1880 B/op    24 allocs/op
BenchmarkPlaygroundNestedValid-28   7199636    166 ns/op        0 B/op     0 allocs/op
```

汇总对比（数值取约值）：

| 场景 | 指标 | gookit | go-playground | 谁更优 |
|------|------|-------:|--------------:|--------|
| Flat 成功 | ns/op | ~2016 | ~470 | go-playground 快约 4.3x |
|           | B/op  | ~1844 | 0 | go-playground 零分配 |
|           | allocs/op | 23 | 0 | go-playground 零分配 |
| Flat 失败 | ns/op | ~2468 | ~747 | go-playground 快约 3.3x |
|           | B/op  | ~3751 | ~692 | go-playground 更省 |
|           | allocs/op | 40 | 13 | go-playground 更省 |
| Nested 成功 | ns/op | ~1959 | ~166 | go-playground 快约 11.8x |
|             | B/op  | ~1880 | 0 | go-playground 零分配 |
|             | allocs/op | 24 | 0 | go-playground 零分配 |

**忠于数据的解读**：

- 在“纯等价规则的校验调用”这一窄口径上，**go-playground/validator 明显更快、且成功路径零分配**。
  原因主要是其调用模型——复用单个 validator 实例 + 结构体元信息深度缓存，校验热路径几乎不分配；
  而 gookit 当前 public `Struct()` API 每轮构建 `Validation` 实例并产生中间数据（safeData 等），
  分配次数与字节数都更高。
- gookit 的 `Struct().Validate()` 已较 v1.x 大幅优化（见 `docs/benchmark-v1-to-v2.md`），
  但与“专注校验、热路径零分配”的 go-playground 相比，在这条窄赛道上仍有差距。
- 反过来，本基准 **没有** 覆盖 gookit 独有的 filter / 绑定 / 一体化 API 带来的工程便利，
  这些价值不体现在 ns/op 上。选型应结合实际需求，而非只看本表。

> 若你在自己环境重新运行得到不同数字，请以本机结果为准（CPU / Go 版本 / 负载都会影响）。
