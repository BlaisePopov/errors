// Package errors provides errors that have stack-traces.
//
// This is particularly useful when you want to understand the
// state of execution when an error was returned unexpectedly.
//
// It provides the type *Error which implements the standard
// Go error interface, so you can use this library interchangably
// with code that is expecting a normal error return.
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
//	    return errors.New(Crashed)
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

var stackBufPool = sync.Pool{
	New: func() any {
		buf := make([]uintptr, MaxStackDepth)
		return &buf
	},
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

// New makes an Error from the given value. If that value is already an
// error then it will be used directly, if not, it will be passed to
// fmt.Errorf("%v"). The stacktrace will point to the line of code that
// called New.
func New(e any) *Error {
	var err error

	switch e := e.(type) {
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	bufPtr := stackBufPool.Get().(*[]uintptr)
	defer stackBufPool.Put(bufPtr)
	length := runtime.Callers(2, (*bufPtr)[:])
	stack := make([]uintptr, length)
	copy(stack, (*bufPtr)[:length])

	var file string
	var line int
	var fnName string
	if length > 0 {
		frame, _ := runtime.CallersFrames(stack[:1]).Next()
		file = frame.File
		line = frame.Line
		fnName = frame.Function
	}

	return &Error{
		Err:     err,
		stack:   stack,
		locFile: file,
		locLine: line,
		locFunc: fnName,
	}
}

// Wrap makes an Error from the given value. If that value is already an *Error
// it will not be wrapped and instead will be returned without modification. If
// that value is already an error then it will be used directly and wrapped.
// Otherwise, the value will be passed to fmt.Errorf("%v") and then wrapped. To
// explicitly wrap an *Error with a new stacktrace use Errorf. The skip
// parameter indicates how far up the stack to start the stacktrace. 0 is from
// the current call, 1 from its caller, etc.
func Wrap(e any, skip int) *Error {
	if e == nil {
		return nil
	}

	var err error

	switch e := e.(type) {
	case *Error:
		return e
	case error:
		err = e
	default:
		err = fmt.Errorf("%v", e)
	}

	bufPtr := stackBufPool.Get().(*[]uintptr)
	defer stackBufPool.Put(bufPtr)
	length := runtime.Callers(2+skip, (*bufPtr)[:])
	stack := make([]uintptr, length)
	copy(stack, (*bufPtr)[:length])
	return &Error{
		Err:   err,
		stack: stack,
	}
}

// WrapPrefix makes an Error from the given value. If that value is already an
// *Error it will not be wrapped and instead will be returned without
// modification. If that value is already an error then it will be used
// directly and wrapped.  Otherwise, the value will be passed to
// fmt.Errorf("%v") and then wrapped. To explicitly wrap an *Error with a new
// stacktrace use Errorf. The prefix parameter is used to add a prefix to the
// error message when calling Error(). The skip parameter indicates how far up
// the stack to start the stacktrace. 0 is from the current call, 1 from its
// caller, etc.
func WrapPrefix(e any, prefix string, skip int) *Error {
	if e == nil {
		return nil
	}

	var inner error
	switch v := e.(type) {
	case *Error:
		inner = v
	case error:
		inner = v
	default:
		inner = fmt.Errorf("%v", v)
	}

	var rpc [1]uintptr
	n := runtime.Callers(2+skip, rpc[:])
	var file string
	var line int
	var fnName string
	if n > 0 {
		frame, _ := runtime.CallersFrames(rpc[:]).Next()
		file = frame.File
		line = frame.Line
		fnName = frame.Function
	}

	return &Error{
		Err:     inner,
		prefix:  prefix,
		locFile: file,
		locLine: line,
		locFunc: fnName,
	}
}

// Errorf creates a new error with the given message. You can use it
// as a drop-in replacement for fmt.Errorf() to provide descriptive
// errors in return values.
func Errorf(format string, a ...any) *Error {
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
