# go-errors/errors

[![Go Reference](https://pkg.go.dev/badge/github.com/go-errors/errors.svg)](https://pkg.go.dev/github.com/go-errors/errors)

**Languages:** [English](README.md) | [Русский](README.ru.md) | [Español](README.es.md) | 中文

`errors` 包为 Go 中的错误添加了堆栈跟踪支持。

它是标准库 `errors` 包的**直接替代品**：只需更改导入路径，所有对 `errors.New`、`errors.Is`、`errors.As`、`errors.Unwrap` 和 `errors.Join` 的现有调用将继续工作——但现在 `errors.New` 及相关函数还会捕获堆栈跟踪。

## 与 `errors`（标准库）的兼容性

| 函数 / 变量                        | 标准库 | 本包      | 备注                                  |
|------------------------------------|--------|----------|---------------------------------------|
| `New(text string) error`           | ✅     | ✅       | 额外捕获堆栈跟踪                      |
| `Is(err, target error) bool`       | ✅     | ✅       | 委托给 `errors.Is`                    |
| `As(err error, target any) bool`   | ✅     | ✅       | 委托给 `errors.As`                    |
| `Unwrap(err error) error`          | ✅     | ✅       | 委托给 `errors.Unwrap`                |
| `Join(errs ...error) error`        | ✅     | ✅       | 委托给 `errors.Join`（Go 1.20+）      |
| `ErrUnsupported`                   | ✅     | ✅       | 重新导出的哨兵值（Go 1.21+）          |

### 扩展 API

| 函数                                              | 描述                                             |
|---------------------------------------------------|--------------------------------------------------|
| `From(v any) *Error`                              | 将任意值包装为带堆栈跟踪的 `*Error`              |
| `Wrap(err error, skip int) error`                 | 用堆栈跟踪包装现有错误                           |
| `WrapPrefix(err error, prefix string, skip int)`  | 用描述性前缀 + 堆栈跟踪包装                      |
| `Errorf(format string, a ...any) error`           | 类似 `fmt.Errorf`，但带堆栈跟踪                  |
| `ParsePanic(text string) (*Error, error)`         | 从 panic 输出重建 `*Error`                       |

## 安装

```bash
go get github.com/go-errors/errors
```

## 快速入门

```go
package main

import (
    "fmt"
    "github.com/go-errors/errors"
)

var ErrNotFound = errors.New("not found")

func findItem(id int) error {
    return errors.WrapPrefix(ErrNotFound, fmt.Sprintf("item %d", id), 0)
}

func main() {
    err := findItem(42)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            // Print full stack trace
            fmt.Println(err.(*errors.Error).ErrorStack())
        }
    }
}
```

## `*Error` 类型

`*Error` 类型实现了标准 `error` 接口，并提供：

- **`Error() string`** — 错误消息（可选前缀）
- **`Unwrap() error`** — 返回被包装的错误
- **`Stack() []byte`** — 格式化的堆栈跟踪（类似 `runtime/debug.Stack()`）
- **`StackFrames() []StackFrame`** — 结构化的栈帧数据
- **`ErrorStack() string`** — 类型 + 消息 + 堆栈跟踪合为一个字符串
- **`TypeName() string`** — 底层错误的类型名称
- **`Location() (file string, line int)`** — 创建错误时的文件和行号
- **`FuncName() string`** — 创建错误时的函数名
- **`Prefix() string`** — 通过 `WrapPrefix` 设置的前缀
- **`Callers() []uintptr`** — 原始程序计数器

## 线程安全

`*Error` 对象在创建后可安全地进行并发读取。
栈帧和格式化的堆栈输出通过 `sync.Once` 延迟计算，
因此对 `StackFrames()` 和 `Stack()` 的并发调用是安全的。

## 基准测试

### 内部基准测试

结果（Windows/amd64，Intel i5-8250U）：

| 操作                    | ns/op | allocs | B/op |
|--------------------------|------:|-------:|-----:|
| `New()`                  |  964  |   3    | 192  |
| `Wrap()`                 |  611  |   2    | 176  |
| `WrapPrefix()`           |  422  |   1    | 144  |
| `Error()`                |    4  |   0    |   0  |
| `StackFrames()` (cached) |    3  |   0    |   0  |
| `Stack()`                | 1659  |  12    | 1248 |
| `ErrorStack()`           | 2267  |  15    | 2208 |
| `From()`                 | 1678  |   2    | 176  |

### 比较基准测试（vs. cockroachdb/errors、juju/errors）

#### New — 叶子错误创建

| 包                      | ns/op  | allocs | B/op |
|------------------------|-------:|-------:|-----:|
| **本包**               |   1903 |   3    |  192 |
| juju/errors            |    738 |   3    |  328 |
| cockroachdb/errors     |   1639 |   7    |  416 |
| go-errors/errors       |   1785 |   4    |  528 |

#### Single Wrap — 包装已有错误

| 包                      | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **本包**               |   471 |   1    |  144 |
| juju/errors            |   774 |   3    |  328 |
| cockroachdb/errors     |  2608 |   7    |  432 |
| go-errors/errors       |    79 |   1    |   80 |

#### Create + Wrap ×5 — 完整错误链

| 包                      | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **本包**               |  5145 |   8    |  928 |
| juju/errors            |  5320 |  18    | 1968 |
| cockroachdb/errors     | 11126 |  42    | 2577 |
| go-errors/errors       |  2496 |  21    | 1224 |

#### Error() — 5 层包装链的字符串格式化

| 包                      | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **本包**               |   417 |   5    |  248 |
| juju/errors            |  2682 |  15    |  408 |
| cockroachdb/errors     | 11333 |  67    | 5928 |
| go-errors/errors       |   255 |   3    |  112 |

#### 堆栈跟踪提取

| 包                      |    ns/op | allocs |   B/op |
|------------------------|--------:|-------:|-------:|
| **本包**               |     685 |   8    |   520  |
| juju/errors            |   4 173  |  31    |  1680  |
| cockroachdb/errors     |  50 990  | 126    | 22585  |
| go-errors/errors       | 861 620  |  70    | 27791  |

#### Unwrap all — 完整链遍历

| 包                      | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **本包**               | 40.5  |   0    |   0  |
| juju/errors            |  6.4  |   0    |   0  |
| cockroachdb/errors     | 86.9  |   0    |   0  |
| go-errors/errors       |  9.0  |   0    |   0  |

### 结论

1. **堆栈跟踪提取是最大优势。** 本包在堆栈跟踪渲染方面比 juju/errors **快 6 倍**，比 cockroachdb/errors **快 74 倍**，比 go-errors/errors **快 1 258 倍**——得益于 `bytes.Buffer` 的零拷贝输出和通过 `sync.Once` 的延迟帧解析。

2. **WrapPrefix 在内存分配方面非常高效。** 单次包装仅产生 **1 次分配 / 144 B**，优于 juju（3/328）和 cockroachdb（7/432）。5 层包装链使用的**分配次数不到任何竞争对手的一半**（8 次分配 vs 18–42 次）。

3. **Error() 字符串格式化速度很快。** 5 层包装链仅需 **417 纳秒**，比 juju 快 6.4 倍，比 cockroachdb 快 27 倍，与不执行前缀拼接的最小化 go-errors/errors（255 纳秒）相比开销适中。

4. **New() 以内存换取速度。** 叶子错误创建使用 **192 B / 3 次分配**——在所有测试包中内存占用最小，同时保持有竞争力的速度。

## 许可证

本包采用 MIT 许可证。详见 [LICENSE.MIT](LICENSE.MIT)。

## 更新日志

* v1.1.0 更新为使用 Go 1.13 的 `errors.Is` 代替 `==`
* v1.2.0 添加了标准库的 `errors.As`
* v1.3.0 *破坏性变更* 更新错误方法以返回 `error` 而非 `*Error`
* v1.4.0 *破坏性变更* 撤销了 v1.3.0 的更改（与 v1.2.0 相同）
* v1.4.1 无代码更改，移除了不必要的 `cover.out` 文件
* v1.4.2 改进了 `ErrorStack()` 的性能
* v1.5.0 添加了 `errors.Join()` 和 `errors.Unwrap()`
* v1.5.1 修复了 Go 1.13–1.19 上的构建问题
* v2.0.0 重大重构：
  - 最低 Go 版本：1.21
  - 添加了 `ErrUnsupported` 哨兵值
  - 修复了 `StackFrames()` 中的竞态条件（现使用 `sync.Once`）
  - 用每个错误的缓存替换了全局 `stackCache`（无内存泄漏）
  - `Wrap()` 和 `WrapPrefix()` 现在捕获完整的堆栈跟踪和位置信息
  - `Is()` 现在完全委托给 `errors.Is`（与标准库兼容的语义）
  - 移除了构建标签拆分文件（`error_1_13.go`、`join_unwrap_1_20.go`）
  - 提升性能：减少堆栈捕获和帧格式化中的内存分配
  - 添加了 `FuncName()` 方法（别名：`LocationFunc()`）
  - 完善的 godoc 注释和 `Example*` 测试函数
