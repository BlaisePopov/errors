package errors

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

const maxSourceCache = 256

var showSourceLines = sync.OnceValue(func() bool {
	return os.Getenv("GOERRORS_HIDE_SOURCE") == ""
})

// StackFrame contains all necessary information to generate a line in a
// call stack trace.
type StackFrame struct {
	// File is the path to the file containing this ProgramCounter.
	File string
	// LineNumber is the line number in that file.
	LineNumber int
	// Name is the name of the function that contains this ProgramCounter.
	Name string
	// Package is the package that contains this function.
	Package string
	// ProgramCounter is the underlying program counter.
	ProgramCounter uintptr

	fn *runtime.Func // cached; avoids repeated FuncForPC lookups
}

// NewStackFrame populates a stack frame object from the program counter.
func NewStackFrame(pc uintptr) StackFrame {
	f := StackFrame{ProgramCounter: pc}
	f.fn = runtime.FuncForPC(pc)
	if f.fn == nil {
		return f
	}
	f.Package, f.Name = packageAndName(f.fn)

	// pc-1 because the program counters we use are usually return addresses,
	// and we want to show the line that corresponds to the function call.
	f.File, f.LineNumber = f.fn.FileLine(pc - 1)
	return f
}

// Func returns the function that contained this frame.
func (f *StackFrame) Func() *runtime.Func {
	if f.fn != nil {
		return f.fn
	}
	if f.ProgramCounter == 0 {
		return nil
	}
	f.fn = runtime.FuncForPC(f.ProgramCounter)
	return f.fn
}

// String returns the stack frame formatted in the same way as go does
// in runtime/debug.Stack().
func (f *StackFrame) String() string {
	var b strings.Builder
	b.Grow(len(f.File) + 32 + len(f.Name) + 64)

	b.WriteString(f.File)
	b.WriteByte(':')
	b.WriteString(strconv.Itoa(f.LineNumber))
	b.WriteString(" (0x")
	b.WriteString(strconv.FormatUint(uint64(f.ProgramCounter), 16))
	b.WriteString(")\n")

	if !showSourceLines() {
		return b.String()
	}

	source, err := f.sourceLine()
	if err != nil {
		return b.String()
	}

	b.WriteByte('\t')
	b.WriteString(f.Name)
	b.WriteString(": ")
	b.WriteString(source)
	b.WriteByte('\n')
	return b.String()
}

// SourceLine gets the line of code (from File and LineNumber) of the
// original source if possible.
func (f *StackFrame) SourceLine() (string, error) {
	source, err := f.sourceLine()
	if err != nil {
		return source, fmt.Errorf("source line: %w", err)
	}
	return source, err
}

var sourceLineCache sync.Map
var sourceCacheCount atomic.Int64

type sourceLineResult struct {
	lines []string
	err   error
}

func (f *StackFrame) sourceLine() (string, error) {
	if f.LineNumber <= 0 {
		return "???", nil
	}

	key := f.File
	if cached, ok := sourceLineCache.Load(key); ok {
		result := cached.(*sourceLineResult)
		if result.err != nil {
			return "", result.err
		}
		if f.LineNumber >= 1 && f.LineNumber <= len(result.lines) {
			return result.lines[f.LineNumber-1], nil
		}
		return "???", nil
	}

	file, err := os.Open(f.File)
	if err != nil {
		if sourceCacheCount.Add(1) > maxSourceCache {
			sourceLineCache.Clear()
			sourceCacheCount.Store(0)
		}
		sourceLineCache.Store(key, &sourceLineResult{err: err})
		return "", err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, string(bytes.Trim(scanner.Bytes(), " \t")))
	}
	if err := scanner.Err(); err != nil {
		if sourceCacheCount.Add(1) > maxSourceCache {
			sourceLineCache.Clear()
			sourceCacheCount.Store(0)
		}
		sourceLineCache.Store(key, &sourceLineResult{err: err})
		return "", err
	}

	if sourceCacheCount.Add(1) > maxSourceCache {
		sourceLineCache.Clear()
		sourceCacheCount.Store(0)
	}
	sourceLineCache.Store(key, &sourceLineResult{lines: lines})

	if f.LineNumber >= 1 && f.LineNumber <= len(lines) {
		return lines[f.LineNumber-1], nil
	}
	return "???", nil
}

// packageAndName splits a runtime.Func's fully-qualified name into the
// package path and the short function name.
func packageAndName(fn *runtime.Func) (string, string) {
	name := fn.Name()
	pkg := ""

	// The name includes the path to the package, which is unnecessary
	// since the file name is already included. Plus, it has center dots.
	// That is, we see
	//
	//	runtime/debug.*T·ptrmethod
	//
	// and want
	//
	//	*T.ptrmethod
	//
	// Since the package path might contain dots (e.g. code.google.com/...),
	// we first remove the path prefix if there is one.
	if lastslash := strings.LastIndex(name, "/"); lastslash >= 0 {
		pkg += name[:lastslash] + "/"
		name = name[lastslash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.ReplaceAll(name, "·", ".")
	return pkg, name
}
