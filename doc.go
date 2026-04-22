// Package errors provides errors with attached stack traces.
//
// This package is a drop-in replacement for the standard library "errors"
// package. It implements the full standard API:
//
//   - [New] creates an error from a text string (with stack trace)
//   - [Is] reports whether any error in the chain matches a target
//   - [As] finds the first error in the chain matching a target type
//   - [Unwrap] returns the result of calling Unwrap on an error
//   - [Join] returns an error wrapping a list of errors
//   - [ErrUnsupported] is the standard sentinel for unsupported operations
//
// Beyond the standard API, this package provides:
//
//   - [From] wraps any value as an *[Error] with a stack trace
//   - [Wrap] wraps an existing error with a stack trace
//   - [WrapPrefix] wraps an error with a descriptive prefix and stack trace
//   - [Errorf] creates a formatted error with a stack trace
//   - [ParsePanic] reconstructs an *[Error] from panic output text
//
// The [Error] type captures a single program counter at each call site. Stack
// traces are built by walking the error chain, resolving frames lazily on first
// access via [Error.StackFrames] or [Error.Stack].
//
// Basic usage:
//
//	import "github.com/BlaisePopov/errors"
//
//	func doWork() error {
//	    if err := riskyCall(); err != nil {
//	        return errors.WrapPrefix(err, "doWork", 0)
//	    }
//	    return nil
//	}
//
//	func main() {
//	    if err := doWork(); err != nil {
//	        if e, ok := err.(*errors.Error); ok {
//	            fmt.Println(e.ErrorStack())
//	        }
//	    }
//	}
package errors
