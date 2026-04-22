package errors

import (
	"bytes"
	baseErrors "errors"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"
)

var ErrUnsupported = baseErrors.ErrUnsupported

const slabSize = 1024

type errorSlab struct {
	buf [slabSize]Error
	idx atomic.Int64
}

var globalSlab atomic.Pointer[errorSlab]

func allocError() *Error {
	for {
		s := globalSlab.Load()
		if s == nil {
			ns := new(errorSlab)
			if !globalSlab.CompareAndSwap(nil, ns) {
				continue
			}
			s = ns
		}
		n := s.idx.Add(1)
		if n <= slabSize {
			return &s.buf[n-1]
		}
		ns := new(errorSlab)
		if globalSlab.CompareAndSwap(s, ns) {
			s = ns
		}
	}
}

type Error struct {
	Err       error
	pc        uintptr
	prefix    string
	errCached uint32
	cachedErr string
	once      sync.Once
	data      *errorData
}

type errorData struct {
	locOnce sync.Once
	locFile string
	locLine int
	locFunc string

	framesOnce sync.Once
	frames     []StackFrame

	stackOnce sync.Once
	stack     []byte

	errStackOnce sync.Once
	errStack     string
}

func (e *Error) ensureData() *errorData {
	e.once.Do(func() {
		if e.data == nil {
			e.data = &errorData{}
		}
	})
	return e.data
}

//go:noinline
func New(text string) error {
	e := allocError()
	e.prefix = text
	e.pc = getCallerPC()
	return e
}

//go:noinline
func From(v any) *Error {
	var err error
	switch v := v.(type) {
	case error:
		err = v
	default:
		err = fmt.Errorf("%v", v)
	}
	e := allocError()
	e.Err = err
	e.pc = getCallerPC()
	return e
}

//go:noinline
func Wrap(err error, skip int) error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	e := allocError()
	e.Err = err
	if skip == 0 {
		e.pc = getCallerPC()
	} else {
		var rpc [1]uintptr
		if runtime.Callers(skip+3, rpc[:]) > 0 {
			e.pc = rpc[0]
		}
	}
	return e
}

//go:noinline
func WrapPrefix(err error, prefix string, skip int) error {
	if err == nil {
		return nil
	}
	e := allocError()
	e.Err = err
	e.prefix = prefix
	if skip == 0 {
		e.pc = getCallerPC()
	} else {
		var rpc [1]uintptr
		if runtime.Callers(skip+3, rpc[:]) > 0 {
			e.pc = rpc[0]
		}
	}
	return e
}

//go:noinline
func Errorf(format string, a ...any) error {
	e := allocError()
	e.Err = fmt.Errorf(format, a...)
	e.pc = getCallerPC()
	return e
}

func Is(err, target error) bool {
	return baseErrors.Is(err, target)
}

func As(err error, target any) bool {
	return baseErrors.As(err, target)
}

func Unwrap(err error) error {
	return baseErrors.Unwrap(err)
}

func Join(errs ...error) error {
	return baseErrors.Join(errs...)
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.prefix
	}
	if e.prefix == "" {
		return e.Err.Error()
	}
	if atomic.LoadUint32(&e.errCached) != 0 {
		return e.cachedErr
	}
	msg := e.prefix + ": " + e.Err.Error()
	e.cachedErr = msg
	atomic.StoreUint32(&e.errCached, 1)
	return msg
}

func (e *Error) resolveLocation() {
	d := e.ensureData()
	d.locOnce.Do(func() {
		if e.pc == 0 {
			return
		}
		fn := runtime.FuncForPC(e.pc - 1)
		if fn == nil {
			return
		}
		d.locFile, d.locLine = fn.FileLine(e.pc - 1)
		d.locFunc = fn.Name()
	})
}

func (e *Error) Stack() []byte {
	d := e.ensureData()
	d.stackOnce.Do(func() {
		frames := e.StackFrames()
		if len(frames) == 0 {
			return
		}
		var buf bytes.Buffer
		buf.Grow(len(frames) * 128)
		for _, frame := range frames {
			buf.WriteString(frame.String())
		}
		d.stack = buf.Bytes()
	})
	return d.stack
}

func (e *Error) Callers() []uintptr {
	pcs := make([]uintptr, 0, 8)
	current := e
	for current != nil {
		if current.pc != 0 {
			pcs = append(pcs, current.pc)
		}
		if inner, ok := current.Err.(*Error); ok {
			current = inner
		} else {
			break
		}
	}
	return pcs
}

func (e *Error) ErrorStack() string {
	d := e.ensureData()
	d.errStackOnce.Do(func() {
		var buf bytes.Buffer
		buf.WriteString(e.TypeName())
		buf.WriteByte(' ')
		buf.WriteString(e.Error())
		buf.WriteByte('\n')
		buf.Write(e.Stack())
		d.errStack = buf.String()
	})
	return d.errStack
}

func (e *Error) StackFrames() []StackFrame {
	d := e.ensureData()
	d.framesOnce.Do(func() {
		if d.frames != nil {
			return
		}
		var chain []*Error
		current := e
		for current != nil {
			chain = append(chain, current)
			if inner, ok := current.Err.(*Error); ok {
				current = inner
			} else {
				break
			}
		}
		d.frames = make([]StackFrame, 0, len(chain))
		for i := len(chain) - 1; i >= 0; i-- {
			pc := chain[i].pc
			if pc != 0 {
				d.frames = append(d.frames, NewStackFrame(pc))
			}
		}
	})
	return d.frames
}

func (e *Error) TypeName() string {
	if _, ok := e.Err.(uncaughtPanic); ok {
		return "panic"
	}
	if e.Err == nil {
		if e.prefix != "" {
			return "string"
		}
		return "nil"
	}
	return reflect.TypeOf(e.Err).String()
}

func (e *Error) Prefix() string {
	return e.prefix
}

func (e *Error) Location() (string, int) {
	e.resolveLocation()
	d := e.data
	if d != nil {
		return d.locFile, d.locLine
	}
	return "", 0
}

func (e *Error) FuncName() string {
	e.resolveLocation()
	d := e.data
	if d != nil {
		return d.locFunc
	}
	return ""
}

func (e *Error) LocationFunc() string {
	return e.FuncName()
}

func (e *Error) Unwrap() error {
	return e.Err
}
