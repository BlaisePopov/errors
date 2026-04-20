package errors

import (
	baseErrors "errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// ErrUnsupported is the standard sentinel indicating that a requested
// operation cannot be performed. It is re-exported from the standard
// library for convenience when using this package as a drop-in replacement.
var ErrUnsupported = baseErrors.ErrUnsupported

// MaxStackDepth is the maximum number of stack frames captured for any error.
// It is safe to read concurrently. If modified, do so before calling
// New, Wrap, WrapPrefix, or From for the first time.
var MaxStackDepth = 32

// captureStack captures up to depth program counters starting skip frames
// above the caller. The skip parameter is relative to the caller of
// captureStack (skip=0 means the caller's caller thanks to the +3 offset).
func captureStack(skip, depth int) []uintptr {
	if depth <= 64 {
		var buf [64]uintptr
		n := runtime.Callers(skip+3, buf[:depth])
		return append([]uintptr(nil), buf[:n]...)
	}
	buf := make([]uintptr, depth)
	n := runtime.Callers(skip+3, buf)
	return buf[:n]
}

// resolveLocation extracts file, line, and function name from the first
// program counter in the stack. Returns zero values if the stack is empty.
func resolveLocation(stack []uintptr) (file string, line int, fnName string) {
	if len(stack) == 0 {
		return "", 0, ""
	}
	fn := runtime.FuncForPC(stack[0] - 1)
	if fn == nil {
		return "", 0, ""
	}
	file, line = fn.FileLine(stack[0] - 1)
	fnName = fn.Name()
	return file, line, fnName
}

// Error is an error with an attached stack trace. It can be used wherever
// the builtin error interface is expected.
type Error struct {
	// Err is the underlying error. It is exported for backward compatibility
	// with code that accesses it directly. Prefer Unwrap for idiomatic usage.
	Err    error
	stack  []uintptr
	prefix string

	locFile string
	locLine int
	locFunc string

	framesOnce sync.Once
	frames     []StackFrame
}

// textError is a simple error that holds a message string, used internally
// by New to avoid importing or depending on the standard errors package
// for leaf errors.
type textError struct {
	msg string
}

func (e *textError) Error() string { return e.msg }

// New returns an error that formats as the given text. It is a drop-in
// replacement for the standard library errors.New and additionally records
// a stack trace at the point it was called.
func New(text string) error {
	stack := captureStack(0, MaxStackDepth)
	file, line, fnName := resolveLocation(stack)

	return &Error{
		Err:     &textError{msg: text},
		stack:   stack,
		locFile: file,
		locLine: line,
		locFunc: fnName,
	}
}

// From wraps the given value as an *Error with a stack trace. If the value
// is already an error it is used directly; otherwise it is converted via
// fmt.Errorf("%v"). The stack trace points to the line of code that called From.
func From(v any) *Error {
	var err error
	switch v := v.(type) {
	case error:
		err = v
	default:
		err = fmt.Errorf("%v", v)
	}

	stack := captureStack(0, MaxStackDepth)
	file, line, fnName := resolveLocation(stack)

	return &Error{
		Err:     err,
		stack:   stack,
		locFile: file,
		locLine: line,
		locFunc: fnName,
	}
}

// Wrap wraps the given error, capturing a stack trace at the call site. If
// err is nil, Wrap returns nil. If err is already an *Error it is returned
// without modification. The skip parameter indicates how far up the stack to
// start the stack trace: 0 is from the current call, 1 from its caller, etc.
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
// the error message. Only the call-site location is captured (not a full
// stack trace), keeping WrapPrefix lightweight for chained wrapping. If err
// is nil, WrapPrefix returns nil. The skip parameter indicates how far up
// the stack to start: 0 is from the current call, 1 from its caller, etc.
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

// Errorf creates a new error with the given message and a stack trace.
// It can be used as a drop-in replacement for fmt.Errorf to provide
// descriptive errors in return values.
func Errorf(format string, a ...any) error {
	return Wrap(fmt.Errorf(format, a...), 1)
}

// Is reports whether any error in err's tree matches target.
//
// This is a drop-in replacement for errors.Is from the standard library.
// The chain is traversed via Unwrap; errors may customize matching by
// implementing an Is(error) bool method.
func Is(err, target error) bool {
	return baseErrors.Is(err, target)
}

// As finds the first error in err's tree that matches target, and if one
// is found, sets target to that error value and returns true. Otherwise,
// it returns false.
//
// This is a drop-in replacement for errors.As from the standard library.
func As(err error, target any) bool {
	return baseErrors.As(err, target)
}

// Unwrap returns the result of calling the Unwrap method on err, if err's
// type contains an Unwrap method returning error. Otherwise, Unwrap
// returns nil.
//
// Unwrap only calls a method of the form "Unwrap() error".
// In particular Unwrap does not unwrap errors returned by [Join].
//
// This is a drop-in replacement for errors.Unwrap from the standard library.
func Unwrap(err error) error {
	return baseErrors.Unwrap(err)
}

// Join returns an error that wraps the given errors. Any nil error values
// are discarded. Join returns nil if every value in errs is nil. The error
// formats as the concatenation of the strings obtained by calling the Error
// method of each element of errs, with a newline between each string.
//
// A non-nil error returned by Join implements the Unwrap() []error method.
//
// This is a drop-in replacement for errors.Join from the standard library.
func Join(errs ...error) error {
	return baseErrors.Join(errs...)
}

// Error returns the underlying error's message. If a prefix was set via
// WrapPrefix, it is prepended as "prefix: message".
func (e *Error) Error() string {
	msg := e.Err.Error()
	if e.prefix != "" {
		msg = e.prefix + ": " + msg
	}
	return msg
}

// Stack returns the call stack formatted the same way as
// runtime/debug.Stack(). The result is computed from cached StackFrames.
func (e *Error) Stack() []byte {
	frames := e.StackFrames()
	if len(frames) == 0 {
		return nil
	}
	var b strings.Builder
	b.Grow(len(frames) * 128)
	for _, frame := range frames {
		b.WriteString(frame.String())
	}
	return []byte(b.String())
}

// Callers returns the raw program counters of the call stack.
func (e *Error) Callers() []uintptr {
	return e.stack
}

// ErrorStack returns a string that contains both the error message and
// the full call stack.
func (e *Error) ErrorStack() string {
	var b strings.Builder
	b.WriteString(e.TypeName())
	b.WriteByte(' ')
	b.WriteString(e.Error())
	b.WriteByte('\n')
	b.Write(e.Stack())
	return b.String()
}

// StackFrames returns the stack frames containing information about the
// call stack. Frames are resolved lazily on first call and cached.
// This method is safe for concurrent use.
func (e *Error) StackFrames() []StackFrame {
	e.framesOnce.Do(func() {
		if e.frames != nil {
			return
		}
		e.frames = make([]StackFrame, len(e.stack))
		for i, pc := range e.stack {
			e.frames[i] = NewStackFrame(pc)
		}
	})
	return e.frames
}

// TypeName returns the type of the underlying error, e.g.
// "*errors.textError". For errors recovered from panics, it returns "panic".
func (e *Error) TypeName() string {
	if _, ok := e.Err.(uncaughtPanic); ok {
		return "panic"
	}
	if e.Err == nil {
		return "nil"
	}
	return reflect.TypeOf(e.Err).String()
}

// Prefix returns the prefix string set via WrapPrefix, or empty if none.
func (e *Error) Prefix() string {
	return e.prefix
}

// Location returns the file path and line number where this error was
// created. If capture-time location info is available (from New, From,
// or WrapPrefix), it is used directly. Otherwise, the first stack
// frame is consulted.
func (e *Error) Location() (string, int) {
	if e.locFile != "" {
		return e.locFile, e.locLine
	}
	frames := e.StackFrames()
	if len(frames) > 0 {
		return frames[0].File, frames[0].LineNumber
	}
	return "", 0
}

// FuncName returns the fully-qualified function name where this error was
// created, or empty if unavailable.
func (e *Error) FuncName() string {
	return e.locFunc
}

// LocationFunc returns the fully-qualified function name where this error
// was created. It is an alias for FuncName, kept for backward compatibility.
func (e *Error) LocationFunc() string {
	return e.locFunc
}

// Is reports whether the current error should be considered a match for
// target. It delegates to the wrapped error so that errors.Is can match
// across *Error boundaries (e.g. From(io.EOF) matches From(io.EOF)).
func (e *Error) Is(target error) bool {
	if target == nil {
		return e == nil
	}
	if t, ok := target.(*Error); ok {
		if t.Err != nil {
			return baseErrors.Is(e.Err, t.Err)
		}
		return e.Err == nil
	}
	return baseErrors.Is(e.Err, target)
}

// Unwrap returns the wrapped error, implementing the standard unwrap
// interface used by errors.Is and errors.As.
func (e *Error) Unwrap() error {
	return e.Err
}
