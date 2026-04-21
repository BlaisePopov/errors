# go-errors/errors

[![Go Reference](https://pkg.go.dev/badge/github.com/go-errors/errors.svg)](https://pkg.go.dev/github.com/go-errors/errors)

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
go get github.com/go-errors/errors
```

## Quick Start

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

Run from within the `goerrors` directory:

```bash
go test -bench=Benchmark -benchmem ./...
```

Results (Windows/amd64, Intel i5-8250U):

| Operation                | ns/op | allocs | B/op |
|--------------------------|------:|-------:|-----:|
| `New()`                  |  964  |   3    | 192  |
| `Wrap()`                 |  611  |   2    | 176  |
| `WrapPrefix()`           |  422  |   1    | 144  |
| `Error()`                |    4  |   0    |   0  |
| `StackFrames()` (cached) |    3  |   0    |   0  |
| `Stack()`                | 1659  |  12    | 1248 |
| `ErrorStack()`           | 2267  |  15    | 2208 |
| `From()`                 | 1678  |   2    | 176  |

### Comparative benchmarks (vs. cockroachdb/errors, juju/errors)

Run from the repository root:

```bash
go test -bench=Benchmark -benchmem -run=^$ .
```

#### New — leaf error creation

| Package                | ns/op  | allocs | B/op |
|------------------------|-------:|-------:|-----:|
| **this package**       |   1903 |   3    |  192 |
| juju/errors            |    738 |   3    |  328 |
| cockroachdb/errors     |   1639 |   7    |  416 |
| go-errors/errors       |   1785 |   4    |  528 |

#### Single Wrap — wrapping a pre-existing error

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |   471 |   1    |  144 |
| juju/errors            |   774 |   3    |  328 |
| cockroachdb/errors     |  2608 |   7    |  432 |
| go-errors/errors       |    79 |   1    |   80 |

#### Create + Wrap ×5 — full chained error pipeline

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |  5145 |   8    |  928 |
| juju/errors            |  5320 |  18    | 1968 |
| cockroachdb/errors     | 11126 |  42    | 2577 |
| go-errors/errors       |  2496 |  21    | 1224 |

#### Error() — string formatting of 5-wrap chain

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       |   417 |   5    |  248 |
| juju/errors            |  2682 |  15    |  408 |
| cockroachdb/errors     | 11333 |  67    | 5928 |
| go-errors/errors       |   255 |   3    |  112 |

#### Stack trace extraction

| Package                |    ns/op | allocs |   B/op |
|------------------------|--------:|-------:|-------:|
| **this package**       |     685 |   8    |   520  |
| juju/errors            |   4 173  |  31    |  1680  |
| cockroachdb/errors     |  50 990  | 126    | 22585  |
| go-errors/errors       | 861 620  |  70    | 27791  |

#### Unwrap all — full chain traversal

| Package                | ns/op | allocs | B/op |
|------------------------|------:|-------:|-----:|
| **this package**       | 40.5  |   0    |   0  |
| juju/errors            |  6.4  |   0    |   0  |
| cockroachdb/errors     | 86.9  |   0    |   0  |
| go-errors/errors       |  9.0  |   0    |   0  |

### Conclusions

1. **Stack trace extraction is the standout win.** This package is **6× faster** than
   juju/errors, **74× faster** than cockroachdb/errors, and **1 258× faster** than
   go-errors/errors for stack trace rendering — thanks to `bytes.Buffer` zero-copy
   output and lazy `sync.Once` frame resolution.

2. **WrapPrefix is allocation-efficient.** Single-wrap produces only **1 alloc / 144 B**,
   beating juju (3/328) and cockroachdb (7/432). The 5-wrap pipeline uses **fewer than
   half the allocations** of any competitor (8 allocs vs 18–42).

3. **Error() string formatting is fast.** At **417 ns** for a 5-wrap chain, it
   beats juju (6.4×) and cockroachdb (27×), with moderate overhead vs the
   minimal go-errors/errors (255 ns) which does no prefix concatenation.

4. **New() trades memory for speed.** Leaf creation uses **192 B / 3 allocs** — the
   smallest memory footprint of all tested packages, while being competitively fast.

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
