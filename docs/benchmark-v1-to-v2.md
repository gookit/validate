# 性能对比 (Benchmark)

本文档对比 `gookit/validate` 三个版本的基准测试（benchmark）结果，展示 v1.5.7 → v1.6.0 → v2.0.0 的性能演进趋势。

## 测试说明

- **对比版本**：
  - `v1.5.7`（旧版基线，模块路径 `github.com/gookit/validate`）
  - `v1.6.0`（v1 优化版，模块路径 `github.com/gookit/validate`）
  - `v2.0.0`（重构版，模块路径 `github.com/gookit/validate/v2`）
- **数据来源文件**：
  - v1.5.7：[`docs/plans/v1.6.0-bench/p0-baseline.txt`](plans/v1.6.0-bench/p0-baseline.txt)
  - v1.6.0：[`docs/plans/v1.6.0-bench/p6-final.txt`](plans/v1.6.0-bench/p6-final.txt)
  - v2.0.0：[`docs/plans/v2.0-final.txt`](plans/v2.0-final.txt)
- **取值口径**：每个 benchmark 在原始输出里跑了多次（v1.5.7 为 3 次，v1.6.0 与 v2.0.0 为 6 次），本文档对同名 benchmark 的所有迭代结果按 `ns/op`、`B/op`、`allocs/op` 分别**取算术平均**。
- **提速倍数算法**：`倍数 = 旧版本 ns/op ÷ 新版本 ns/op`（>1 表示新版本更快）；百分比变化以旧版本为基准，正值表示更快/更省。

### 测试环境

三个文件的硬件与系统环境完全一致：

| 项 | 值 |
|----|----|
| goos | linux |
| goarch | amd64 |
| cpu | Intel(R) Core(TM) i7-14700KF |
| 并行度 | 28（benchmark 后缀 `-28`，即 `GOMAXPROCS=28`） |
| pkg | v1.x: `github.com/gookit/validate`；v2: `github.com/gookit/validate/v2` |

> 说明：原始输出文件头部未记录 Go 版本，故此处不标注；除模块路径因 v2 主版本升级而不同外，其余环境一致，可放心横向对比。

## 主对比表（三版本共有的 6 个 benchmark）

下列 6 个 benchmark 在三个版本中均存在，是主要对比对象。数值均为多次迭代的平均值。

### 耗时 ns/op（越小越快）

| Benchmark | v1.5.7 | v1.6.0 | v2.0.0 | v2.0 vs v1.5.7 | v2.0 vs v1.6.0 |
|-----------|-------:|-------:|-------:|:--------------:|:--------------:|
| MapValidate          |  9606.67 | 3314.17 | 3426.33 | **2.80x 更快**（-64.3%） | 0.97x 略慢（+3.4%） |
| StructFirstParse     |  8577.00 |  754.33 |  480.48 | **17.85x 更快**（-94.4%） | **1.57x 更快**（-36.3%） |
| StructFlat           | 10404.67 | 2294.83 | 1904.00 | **5.46x 更快**（-81.7%） | **1.21x 更快**（-17.0%） |
| StructNested         | 10143.67 | 2140.83 | 1757.50 | **5.77x 更快**（-82.7%） | **1.22x 更快**（-17.9%） |
| StructSliceOfStruct  | 17285.00 | 10785.17 | 11092.33 | **1.56x 更快**（-35.8%） | 0.97x 略慢（+2.8%） |
| ValValue             |   717.73 |  724.62 |  743.50 | 0.97x 略慢（+3.6%） | 0.97x 略慢（+2.6%） |

### 内存分配 B/op（越小越省）

| Benchmark | v1.5.7 | v1.6.0 | v2.0.0 | v2.0 vs v1.5.7 | v2.0 vs v1.6.0 |
|-----------|-------:|-------:|-------:|:--------------:|:--------------:|
| MapValidate          | 17918.0 | 3746.8 | 3900.7 | -78.2% | +4.1%（略增） |
| StructFirstParse     | 17392.0 | 2408.0 | 1096.0 | -93.7% | -54.5% |
| StructFlat           | 18353.3 | 3154.8 | 1831.2 | -90.0% | -42.0% |
| StructNested         | 17936.0 | 3032.0 | 1880.0 | -89.5% | -38.0% |
| StructSliceOfStruct  | 24062.0 | 10070.0 | 10424.0 | -56.7% | +3.5%（略增） |
| ValValue             |   494.7 |  494.0 |  527.3 | +6.6%（略增） | +6.7%（略增） |

> 注：B/op 列负号表示内存占用**下降**（更省），正号表示**上升**（更费）。

### 分配次数 allocs/op（越小越好）

| Benchmark | v1.5.7 | v1.6.0 | v2.0.0 | v2.0 vs v1.5.7 | v2.0 vs v1.6.0 |
|-----------|-------:|-------:|-------:|:--------------:|:--------------:|
| MapValidate          | 124 | 72 | 72 | -41.9% | 持平 |
| StructFirstParse     | 125 | 27 | 16 | -87.2% | -40.7% |
| StructFlat           | 132 | 34 | 23 | -82.6% | -32.4% |
| StructNested         | 132 | 34 | 24 | -81.8% | -29.4% |
| StructSliceOfStruct  | 260 | 208 | 208 | -20.0% | 持平 |
| ValValue             |  11 | 11 | 11 | 持平 | 持平 |

## 版本特有 benchmark

以下 benchmark 并非三版本都有，无法横向构成完整对比，故单独列出，不进入主对比表。

### FieldSuccess / FieldSuccessParallel（仅 v1.5.7、v2.0.0）

这两个用例衡量单字段最优路径的极限开销，量级在亚纳秒级（受编译器内联/优化影响很大，主要作参考）。**v1.6.0 缺少**该用例，故不进主表。

| Benchmark | v1.5.7 (ns/op) | v2.0.0 (ns/op) |
|-----------|---------------:|---------------:|
| FieldSuccess         | 0.8327 | 0.8250 |
| FieldSuccessParallel | 0.1117 | 0.1142 |

两版本数值基本持平（差异在测量噪声范围内），说明 v2 在单字段最优路径上未引入额外开销。

### FactoryStructReuse（仅 v2.0.0）

该用例衡量 v2 **新增的 Factory 复用能力**（复用解析后的校验器以摊销结构体解析成本），v1.x 不具备此能力，故只在 v2.0.0 中存在，无对比对象。

| Benchmark | v2.0.0 ns/op | v2.0.0 B/op | v2.0.0 allocs/op |
|-----------|-------------:|------------:|-----------------:|
| FactoryStructReuse | 1654.67 | 916.3 | 11 |

对照同版本的 `StructFlat`（1904.00 ns/op、1831.2 B/op、23 allocs/op）可见：在复用场景下，耗时进一步降低、内存约减半、分配次数从 23 降到 11，体现了 Factory 复用对重复校验同一结构体类型的优化价值。

### RV-native 优化后（v2.x 持续优化）

v2.0.0 之后，针对 v2 仍带的「校验器入参 `any` + 二次反射」做了一轮 **RV-native 重构**：把 ~60+ 无状态内建校验器搬入 `internal/validators`（入参为封装的 `reflect.Value`，载体 `internal/fieldval.FieldValue`）、热路径直调 internal RV 版去二次反射、新增 `func(FieldCtx) bool` 自定义校验器形态（typed、免 `reflect.Call`，与旧 `func(val any,...) bool` 共存），并在取值链端到端去装箱（载体懒构造持 RV、`Src` 懒装箱、`valueCompare`/`required` 纯 RV）。

flat-valid 成功路径三入口最新实测（同 i7-14700KF，go1.25.10，`_examples/bench-vs-goplayground`）：

| 入口 | allocs/op | B/op | ns/op |
|---|---:|---:|---:|
| `Struct(&u).Validate()` | 7 | ~769 | ~1320 |
| `Check(&u)` | 6 | ~405 | ~1460 |
| `CheckErr(&u)` | **0** | **0** | ~1155 |
| go-playground `Struct(&u)`（v10.30.3，对照） | 8 | ~129 | ~750 |

要点：

- **只有 `CheckErr` 达 0 alloc**（端到端去装箱的成果，原为 3 allocs）。`Check` 因要收集 `safeData`（供 `SafeData()`/`BindStruct`，每个通过字段须装箱进 `map[string]any`）仍为 6 allocs（功能取舍）。
- **struct 源只读入口自动免收集**：`Struct().Validate()`/`ValidateErr()`/`ValidateE()`（只返回 bool/error、不暴露 safeData）现默认走 `skipCollect` 快路径，自动跳过 safeData 收集装箱，`Struct(&u).Validate()` 从 12 → **7 allocs**（写回 `UpdateSource` 开启时跨字段仍正确；`Check`/`ValidateR` 经 `needCollect` 强制收集，不受影响；map/form 源不跳过）。
- 对照 go-playground（本基准 flat-valid 实测 8 allocs、nested 4 allocs）：gookit `CheckErr` 的 alloc(0) 低于其 8，但 go-playground 的 **ns/op 仍更快**（~750 vs ~1155）。
- 设计与实施细节见 [`docs/perf/rv-native-validators-rfc.md`](perf/rv-native-validators-rfc.md)。

## 结论

数据呈现的演进趋势（忠于原始数据）：

- **v1.5.7 → v1.6.0 是一次大幅优化**：Struct/Map 类用例耗时、内存、分配次数全面、显著下降（如 `StructFirstParse` 提速超 11 倍、`StructFlat`/`StructNested` 提速约 4.5～4.7 倍）。
- **v2.0.0 相对 v1.5.7 整体大幅领先**：6 个共有用例中有 5 个更快，`StructFirstParse` 累计提速约 **17.85x**，`StructFlat`/`StructNested` 约 5.5～5.8x，内存与分配次数也大幅下降。
- **v2.0.0 相对 v1.6.0 的进一步收益主要集中在结构体解析路径**：
  - `StructFirstParse` 再提速 **1.57x**，内存与分配次数继续大幅下降（B/op -54.5%、allocs -40.7%）；
  - `StructFlat`/`StructNested` 再提速约 1.21～1.22x，内存与分配次数继续下降。
- **少数用例 v2.0 相对 v1.6.0 略有回退（如实记录）**：
  - `MapValidate`：耗时 +3.4%、内存 +4.1%（分配次数持平）；
  - `StructSliceOfStruct`：耗时 +2.8%、内存 +3.5%（分配次数持平）；
  - `ValValue`：耗时 +2.6%、内存 +6.7%（分配次数持平）。
  - 这些回退幅度较小（均在个位数百分比），属于重构后在部分路径上的轻微代价，整体不影响 v2 的领先态势。
- **v2 新增的 `FactoryStructReuse`** 为重复校验同一结构体类型提供了更优路径（相比同版本 `StructFlat` 更快且内存/分配更少），是 v2 在工程化复用场景下的重要增强。

> 总体而言，从 v1.5.7 到 v2.0.0，库在结构体与 Map 校验的核心路径上实现了数倍级别的性能提升与内存/分配的大幅下降；v2 相对 v1.6.0 主要在结构体解析路径上继续优化，个别用例存在小幅回退。

## 与 go-playground/validator 的对照（#215）

针对 issue [#215](https://github.com/gookit/validate/issues/215)，另补了一份 **gookit/validate vs go-playground/validator** 的对照 benchmark，置于独立 module（不影响主仓库依赖）：

- 目录：[`_examples/bench-vs-goplayground/`](../_examples/bench-vs-goplayground/)（独立 `go.mod`，`replace` 指向本地 validate）。
- 运行：`cd _examples/bench-vs-goplayground && go test -bench . -benchmem -run '^$'`。
- 内容：对“等价规则的校验调用”分扁平成功/失败、嵌套成功三组场景对照，附公平性声明（两库特性/语义不同）与本机结果，详见该目录 `README.md`。
- **差距根因分析与优化清单**：见 [`docs/perf/gookit-vs-goplayground-gap-analysis.md`](perf/gookit-vs-goplayground-gap-analysis.md)（23 allocs 归因、go-playground 0 分配原因、safeData/filter flag 实测、Factory 复用实测、优化排序）。
