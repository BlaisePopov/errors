package errors

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
)

// A StackFrame contains all necessary information about to generate a line
// in a callstack.
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
}

// NewStackFrame populates a stack frame object from the program counter.
func NewStackFrame(pc uintptr) (frame StackFrame) {

	frame = StackFrame{ProgramCounter: pc}
	if frame.Func() == nil {
		return
	}
	frame.Package, frame.Name = packageAndName(frame.Func())

	// pc -1 because the program counters we use are usually return addresses,
	// and we want to show the line that corresponds to the function call
	frame.File, frame.LineNumber = frame.Func().FileLine(pc - 1)
	return

}

// Func returns the function that contained this frame.
func (frame *StackFrame) Func() *runtime.Func {
	if frame.ProgramCounter == 0 {
		return nil
	}
	return runtime.FuncForPC(frame.ProgramCounter)
}

// String returns the stackframe formatted in the same way as go does
// in runtime/debug.Stack()
func (frame *StackFrame) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%d (0x%x)\n", frame.File, frame.LineNumber, frame.ProgramCounter)

	source, err := frame.sourceLine()
	if err != nil {
		return b.String()
	}

	fmt.Fprintf(&b, "\t%s: %s\n", frame.Name, source)
	return b.String()
}

// SourceLine gets the line of code (from File and Line) of the original source if possible.
func (frame *StackFrame) SourceLine() (string, error) {
	source, err := frame.sourceLine()
	if err != nil {
		return source, New(err)
	}
	return source, err
}

var sourceLineCache sync.Map

type sourceLineResult struct {
	lines []string
	err   error
}

func (frame *StackFrame) sourceLine() (string, error) {
	if frame.LineNumber <= 0 {
		return "???", nil
	}

	key := frame.File
	if cached, ok := sourceLineCache.Load(key); ok {
		result := cached.(*sourceLineResult)
		if result.err != nil {
			return "", result.err
		}
		lines := result.lines
		if frame.LineNumber >= 1 && frame.LineNumber <= len(lines) {
			return lines[frame.LineNumber-1], nil
		}
		return "???", nil
	}

	file, err := os.Open(frame.File)
	if err != nil {
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
		sourceLineCache.Store(key, &sourceLineResult{err: err})
		return "", err
	}

	sourceLineCache.Store(key, &sourceLineResult{lines: lines})

	if frame.LineNumber >= 1 && frame.LineNumber <= len(lines) {
		return lines[frame.LineNumber-1], nil
	}
	return "???", nil
}

func packageAndName(fn *runtime.Func) (string, string) {
	name := fn.Name()
	pkg := ""

	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//
	//	runtime/debug.*T·ptrmethod
	//
	// and want
	//
	//	*T.ptrmethod
	//
	// Since the package path might contains dots (e.g. code.google.com/...),
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
