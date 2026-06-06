// Command bench-vs-goplayground 是一个独立 module，仅承载 gookit/validate 与
// go-playground/validator 的对照 benchmark（见 bench_test.go）。
//
// 运行：
//
//	cd _examples/bench-vs-goplayground
//	go test -bench . -benchmem -run '^$'
//
// main() 仅为让本 main 包可 `go build`，无实际逻辑；对照逻辑全部在 *_test.go。
package main

func main() {}
