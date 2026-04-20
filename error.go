// Package errors provides errors that have stack-traces.
//
// This is particularly useful when you want to understand the
// state of execution when an error was returned unexpectedly.
//
// It provides the type *Error which implements the standard
// Go error interface, so you can use this library interchangably
// with code that is expecting a normal error return.
//
// This package is a drop-in replacement for the standard library
// "errors" package. You can swap imports:
//
//	import "errors"        →  import "github.com/go-errors/errors"
//
// The New function matches the stdlib signature exactly.
// Additional functions like Wrap, WrapPrefix, Errorf, From, and
// ParsePanic provide extra functionality beyond the stdlib.
//
// For example:
//
//	package crashy
//
//	import "github.com/go-errors/errors"
//
//	var Crashed = errors.Errorf("oh dear")
//
//	func Crash() error {
//	    return errors.Wrap(Crashed, 0)
//	}
//
// This can be called as follows:
//
//	package main
//
//	import (
//	    "crashy"
//	    "fmt"
//	    "github.com/go-errors/errors"
//	)
//
//	func main() {
//	    err := crashy.Crash()
//	    if err != nil {
//	        if errors.Is(err, crashy.Crashed) {
//	            fmt.Println(err.(*errors.Error).ErrorStack())
//	        } else {
//	            panic(err)
//	        }
//	    }
//	}
package errors

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// MaxStackDepth is the maximum number of stackframes on any error.
// It is safe to read concurrently. If modified, do so before calling
// New, Wrap, or WrapPrefix for the first time.
var MaxStackDepth = 32

func captureStack(skip, depth int) []uintptr {
	if depth <= 64 {
		var buf [64]uintptr
		length := runtime.Callers(skip+3, buf[:depth])
		stack := make([]uintptr, length)
		copy(stack, buf[:length])
		return stack
	}
	buf := make([]uintptr, depth)
	length := runtime.Callers(skip+3, buf)
	stack := make([]uintptr, length)
	copy(stack, buf[:length])
	return stack
}

// Error is an error with an attached stacktrace. It can be used
// wherever the builtin error interface is expected.
type Error struct {
	Err    error
	stack  []uintptr
	frames []StackFrame
	prefix string

	locFile string
	locLine int
	locFunc string
}

type msgError struct {
	msg string
}

func (e *msgError) Error() string { return e.msg }

// New returns an error that formats as the given text. It is a drop-in
// replacement for the standard library errors.New and additionally records
// a stacktrace at the point it was called.
func New(text string) error {
	stack := captureStack(0, MaxStackDepth)

	var file string
	var line int
	var fnName string
	if len(stack) > 0 {
		fn := runtime.FuncForPC(stack[0] - 1)
		if fn != nil {
			file, line = fn.FileLine(stack[0] - 1)
			fnName = fn.Name()
		}
	}

	return &Error{
		Err:     &msgError{msg: text},
		stack:   stack,
		locFile: file,
		locLine: line,
		locFunc: fnName,
	}
}

// From wraps the given value as an *Error with a stacktrace. If the value is
// already an error it is used directly; otherwise it is converted via
// fmt.Errorf("%v"). The stacktrace points to the line of code that called From.
func From(e any) *Error {
	var err error

	switch e := e.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	stack := captureStack(0, MaxStackDepth)

	var file string
	var line int
	var fnName string
	if len(stack) > 0 {
		fn := runtime.FuncForPC(stack[0] - 1)
		if fn != nil {
			file, line = fn.FileLine(stack[0] - 1)
			fnName = fn.Name()
		}
	}

	return &Error{
		Err:     err,
		stack:   stack,
		locFile: file,
		locLine: line,
		locFunc: fnName,
	}
}

// Wrap wraps the given error, capturing a stacktrace at the call site. If err
// is already an *Error it is returned without modification. The skip parameter
// indicates how far up the stack to start the stacktrace: 0 is from the
// current call, 1 from its caller, etc.
func Wrap(err error, skip int) error {
	if err == nil {
		return nil
	}

	if e, ok := err.(*Error); ok {
		return e
	}

	stack := captureStack(skip, MaxStackDepth)
	return &Error{
		Err:   err,
		stack: stack,
	}
}

// WrapPrefix wraps the given error with a prefix string that is prepended to
// the error message. If err is already an *Error it is used as the inner error
// (not returned as-is). The skip parameter indicates how far up the stack to
// start the stacktrace: 0 is from the current call, 1 from its caller, etc.
func WrapPrefix(err error, prefix string, skip int) error {
	if err == nil {
		return nil
	}

	var rpc [1]uintptr
	n := runtime.Callers(2+skip, rpc[:])
	var file string
	var line int
	var fnName string
	if n > 0 {
		fn := runtime.FuncForPC(rpc[0] - 1)
		if fn != nil {
			file, line = fn.FileLine(rpc[0] - 1)
			fnName = fn.Name()
		}
	}

	return &Error{
		Err:     err,
		prefix:  prefix,
		locFile: file,
		locLine: line,
		locFunc: fnName,
	}
}

// Errorf creates a new error with the given message and a stacktrace. You can
// use it as a drop-in replacement for fmt.Errorf() to provide descriptive
// errors in return values.
func Errorf(format string, a ...any) error {
	return Wrap(fmt.Errorf(format, a...), 1)
}

// Error returns the underlying error's message.
func (err *Error) Error() string {
	msg := err.Err.Error()
	if err.prefix != "" {
		msg = err.prefix + ": " + msg
	}
	return msg
}

// Stack returns the callstack formatted the same way that go does
// in runtime/debug.Stack()
var stackCache sync.Map

func (err *Error) Stack() []byte {
	if cached, ok := stackCache.Load(err); ok {
		return cached.([]byte)
	}
	buf := bytes.NewBuffer(make([]byte, 0, len(err.stack)*128))
	for _, frame := range err.StackFrames() {
		buf.WriteString(frame.String())
	}
	b := buf.Bytes()
	stackCache.Store(err, b)
	return b
}

// Callers returns the stack of callers.
func (err *Error) Callers() []uintptr {
	return err.stack
}

// ErrorStack returns a string that contains both the
// error message and the callstack.
func (err *Error) ErrorStack() string {
	var b strings.Builder
	b.WriteString(err.TypeName())
	b.WriteByte(' ')
	b.WriteString(err.Error())
	b.WriteByte('\n')
	b.Write(err.Stack())
	return b.String()
}

// StackFrames returns an array of frames containing information about the
// stack.
func (err *Error) StackFrames() []StackFrame {
	if err.frames == nil {
		err.frames = make([]StackFrame, len(err.stack))

		for i, pc := range err.stack {
			err.frames[i] = NewStackFrame(pc)
		}
	}

	return err.frames
}

// TypeName returns the type this error. e.g. *errors.stringError.
func (err *Error) TypeName() string {
	if _, ok := err.Err.(uncaughtPanic); ok {
		return "panic"
	}
	if err.Err == nil {
		return "nil"
	}
	return reflect.TypeOf(err.Err).String()
}

func (err *Error) Prefix() string {
	return err.prefix
}

func (err *Error) Location() (string, int) {
	if err.locFile != "" {
		return err.locFile, err.locLine
	}
	frames := err.StackFrames()
	if len(frames) > 0 {
		return frames[0].File, frames[0].LineNumber
	}
	return "", 0
}

func (err *Error) LocationFunc() string {
	if err.locFunc != "" {
		return err.locFunc
	}
	return ""
}

// Unwrap returns the wrapped error (implements api for As function).
func (err *Error) Unwrap() error {
	return err.Err
}
