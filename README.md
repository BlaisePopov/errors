# BlaisePopov/errors

[![Go Reference](https://pkg.go.dev/badge/github.com/BlaisePopov/errors.svg)](https://pkg.go.dev/github.com/BlaisePopov/errors)

**Languages:** English | [Русский](README.ru.md) | [Español](README.es.md) | [中文](README.zh.md)

Package `errors` adds stack trace support to errors in Go.

It is a **drop-in replacement** for the standard library `errors` package:
simply swap the import path and every existing call to `errors.New`,
`errors.Is`, `errors.As`, `errors.Unwrap`, and `errors.Join` continues to
work — but now `errors.New` and friends also capture a stack trace.

## Compatibility with `errors` (stdlib)

| Function / Variable                | stdlib | this package | Notes                                 |
|------------------------------------|--------|--------------|---------------------------------------|
| `New(text string) error`           | ✅     | ✅           | Additionally captures stack trace     |
| `Is(err, target error) bool`       | ✅     | ✅           | Delegates to `errors.Is`              |
| `As(err error, target any) bool`   | ✅     | ✅           | Delegates to `errors.As`              |
| `Unwrap(err error) error`          | ✅     | ✅           | Delegates to `errors.Unwrap`          |
| `Join(errs ...error) error`        | ✅     | ✅           | Delegates to `errors.Join` (Go 1.20+) |
| `ErrUnsupported`                   | ✅     | ✅           | Re-exported sentinel (Go 1.21+)       |

### Extended API

| Function                                         | Description                                         |
|--------------------------------------------------|-----------------------------------------------------|
| `From(v any) *Error`                             | Wrap any value as `*Error` with stack trace          |
| `Wrap(err error, skip int) error`                | Wrap existing error with stack trace                 |
| `WrapPrefix(err error, prefix string, skip int)` | Wrap with descriptive prefix + stack trace           |
| `Errorf(format string, a ...any) error`          | Like `fmt.Errorf` but with stack trace               |
| `ParsePanic(text string) (*Error, error)`        | Reconstruct `*Error` from panic output               |

## Installation

```bash
go get github.com/BlaisePopov/errors
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/BlaisePopov/errors"
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

## The `*Error` Type

The `*Error` type implements the standard `error` interface and provides:

- **`Error() string`** — error message (with optional prefix)
- **`Unwrap() error`** — returns the wrapped error
- **`Stack() []byte`** — formatted stack trace (like `runtime/debug.Stack()`)
- **`StackFrames() []StackFrame`** — structured stack frame data
- **`ErrorStack() string`** — type + message + stack trace in one string
- **`TypeName() string`** — type name of the underlying error
- **`Location() (file string, line int)`** — file and line where error was created
- **`FuncName() string`** — function name where error was created
- **`Prefix() string`** — prefix set via `WrapPrefix`
- **`Callers() []uintptr`** — raw program counters

## Thread Safety

`*Error` objects are safe for concurrent read access after creation.
Stack frames and formatted stack output are computed lazily with `sync.Once`,
making concurrent calls to `StackFrames()` and `Stack()` safe.

## Benchmarks

### Internal benchmarks

Results (Windows/amd64, Intel i5-8250U):

| Operation                | ns/op | allocs | B/op |
|--------------------------|------:|-------:|-----:|
| `New()`                  |  195  |   1    |  96  |
| `Wrap()`                 |  218  |   1    |  96  |
| `WrapPrefix()`           |  209  |   1    |  96  |
| `Error()`                |    4  |   0    |   0  |
| `StackFrames()` (cached) |    5  |   0    |   0  |
| `Stack()` (cached)       |    5  |   0    |   0  |
| `ErrorStack()` (cached)  |    5  |   0    |   0  |
| `From()`                 |  219  |   1    |  96  |

### Comparative benchmarks (vs. cockroachdb/errors, juju/errors)

#### New — leaf error creation

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |   210 |   1    |  96  |
| juju/errors            |   689 |   3    | 328  |
| cockroachdb/errors     |  1553 |   7    | 416  |
| go-errors/errors       |   894 |   4    | 528  |

#### Single Wrap — wrapping a pre-existing error

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |   220 |   1    |  96  |
| juju/errors            |   778 |   3    | 328  |
| cockroachdb/errors     |  1836 |   7    | 432  |
| go-errors/errors       |    81 |   1    |  80  |

#### Create + Wrap ×5 — full chained error pipeline

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |  1161 |   6    | 576  |
| juju/errors            |  6088 |  18    | 1968 |
| cockroachdb/errors     | 12461 |  42    | 2577 |
| go-errors/errors       |  2364 |  21    | 1224 |

#### Error() — string formatting of 5-wrap chain

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |   3.4 |   0    |   0  |
| juju/errors            |  3059 |  15    | 408  |
| cockroachdb/errors     | 11996 |  67    | 5945 |
| go-errors/errors       |   286 |   3    | 112  |

#### Stack trace extraction

| Package                |     ns/op | allocs |    B/op |
|------------------------|---------:|-------:|--------:|
| **this package**       |      5.6 |   0    |      0  |
| juju/errors            |     4422 |  31    |   1680  |
| cockroachdb/errors     |    56594 | 126    |  22604  |
| go-errors/errors       |   555963 |  70    |  27790  |

#### Unwrap all — full chain traversal

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |  35.4 |   0    |   0  |
| juju/errors            |   6.5 |   0    |   0  |
| cockroachdb/errors     |  72.9 |   0    |   0  |
| go-errors/errors       |   8.5 |   0    |   0  |

## License

This package is licensed under the MIT license. See [LICENSE.MIT](LICENSE.MIT) for details.

## Changelog

* v1.1.0 Updated to use Go 1.13's `errors.Is` instead of `==`
* v1.2.0 Added `errors.As` from the standard library
* v1.3.0 *BREAKING* Updated error methods to return `error` instead of `*Error`
* v1.4.0 *BREAKING* Reverted v1.3.0 changes (identical to v1.2.0)
* v1.4.1 No code change, removed unnecessary `cover.out` file
* v1.4.2 Performance improvement to `ErrorStack()`
* v1.5.0 Added `errors.Join()` and `errors.Unwrap()`
* v1.5.1 Fixed build on Go 1.13–1.19
* v2.0.0 Major refactoring:
  - Minimum Go version: 1.21
  - Added `ErrUnsupported` sentinel
  - Fixed race condition in `StackFrames()` (now uses `sync.Once`)
  - Replaced global `stackCache` with per-error caching (no memory leak)
  - `Wrap()` and `WrapPrefix()` now capture full stack traces and location info
  - `Is()` now delegates purely to `errors.Is` (stdlib-compatible semantics)
  - Removed build-tag split files (`error_1_13.go`, `join_unwrap_1_20.go`)
  - Improved performance: fewer allocations in stack capture and frame formatting
  - Added `FuncName()` method (alias: `LocationFunc()`)
  - Comprehensive godoc comments and `Example*` test functions
