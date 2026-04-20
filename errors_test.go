package errors

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkStackFormat(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		func() {
			defer func() {
				err := recover()
				if err != 'a' {
					b.Fatal(err)
				}

				e := Errorf("hi").(*Error)
				_ = string(e.Stack())
			}()

			a()
		}()
	}
}

// ---------------------------------------------------------------------------
// Is / As
// ---------------------------------------------------------------------------

func TestIs_nil(t *testing.T) {
	if Is(nil, io.EOF) {
		t.Errorf("nil should not match io.EOF")
	}
}

func TestIs_identical(t *testing.T) {
	if !Is(io.EOF, io.EOF) {
		t.Errorf("io.EOF should match io.EOF")
	}
}

func TestIs_wrappedSource(t *testing.T) {
	if !Is(From(io.EOF), io.EOF) {
		t.Errorf("From(io.EOF) should match io.EOF via Unwrap")
	}
}

func TestIs_doubleWrapped(t *testing.T) {
	if !Is(From(io.EOF), From(io.EOF)) {
		t.Errorf("From(io.EOF) should match From(io.EOF)")
	}
}

func TestIs_differentErrors(t *testing.T) {
	if Is(io.EOF, fmt.Errorf("io.EOF")) {
		t.Errorf("io.EOF should not match fmt.Errorf(\"io.EOF\")")
	}
}

func TestIs_customIsMethod(t *testing.T) {
	t.Parallel()
	custErr := errorWithCustomIs{
		Key: "TestForFun",
		Err: io.EOF,
	}

	shouldMatch := errorWithCustomIs{
		Key: "TestForFun",
	}
	shouldNotMatch := errorWithCustomIs{Key: "notOk"}

	tests := []struct {
		name string
		err  error
		target error
		want bool
	}{
		{"direct match", custErr, shouldMatch, true},
		{"direct no match", custErr, shouldNotMatch, false},
		{"wrap target match", custErr, Wrap(shouldMatch, 0), false},     // stdlib Is does NOT unwrap target
		{"wrap target no match", custErr, Wrap(shouldNotMatch, 0), false},
		{"wrap source match", Wrap(custErr, 0), shouldMatch, true},
		{"wrap source no match", Wrap(custErr, 0), shouldNotMatch, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Is(tt.err, tt.target)
			if got != tt.want {
				t.Errorf("Is(%v, %v) = %v, want %v", tt.err, tt.target, got, tt.want)
			}
		})
	}
}

func TestAs(t *testing.T) {
	var errStrIn errorString = "TestForFun"

	var errStrOut errorString
	if As(errStrIn, &errStrOut) {
		if errStrOut != "TestForFun" {
			t.Errorf("direct errStr value is not returned")
		}
	} else {
		t.Errorf("direct errStr is not returned")
	}

	errStrOut = ""
	err := Wrap(errStrIn, 0)
	if As(err, &errStrOut) {
		if errStrOut != "TestForFun" {
			t.Errorf("wrapped errStr value is not returned")
		}
	} else {
		t.Errorf("wrapped errStr is not returned")
	}
}

// ---------------------------------------------------------------------------
// Stack traces
// ---------------------------------------------------------------------------

func TestStackFormat(t *testing.T) {
	defer func() {
		err := recover()
		if err != 'a' {
			t.Fatal(err)
		}

		e, expected := Errorf("hi").(*Error), callers()

		bs := [][]uintptr{e.stack, expected}

		if err := compareStacks(bs[0], bs[1]); err != nil {
			t.Errorf("Stack didn't match")
			t.Error(err.Error())
		}

		stack := string(e.Stack())

		if !strings.Contains(stack, "a: b(5)") {
			t.Errorf("Stack trace does not contain source line: 'a: b(5)'")
			t.Error(stack)
		}
		if !strings.Contains(stack, "errors_test.go:") {
			t.Errorf("Stack trace does not contain file name: 'errors_test.go:'")
			t.Error(stack)
		}
	}()

	a()
}

func TestSkipWorks(t *testing.T) {
	defer func() {
		err := recover()
		if err != 'a' {
			t.Fatal(err)
		}

		bs := [][]uintptr{Wrap(fmt.Errorf("hi"), 2).(*Error).stack, callersSkip(2)}

		if err := compareStacks(bs[0], bs[1]); err != nil {
			t.Errorf("Stack didn't match")
			t.Error(err.Error())
		}
	}()

	a()
}

// ---------------------------------------------------------------------------
// New / From
// ---------------------------------------------------------------------------

func TestNew(t *testing.T) {
	err := New("foo")

	if err.Error() != "foo" {
		t.Errorf("Wrong message")
	}

	fromErr := From(fmt.Errorf("foo"))

	if fromErr.Error() != "foo" {
		t.Errorf("Wrong message")
	}

	bs := [][]uintptr{New("foo").(*Error).stack, callers()}

	if err := compareStacks(bs[0], bs[1]); err != nil {
		t.Errorf("Stack didn't match")
		t.Error(err.Error())
	}

	e := err.(*Error)
	if e.ErrorStack() != e.TypeName()+" "+e.Error()+"\n"+string(e.Stack()) {
		t.Errorf("ErrorStack is in the wrong format")
	}
}

// ---------------------------------------------------------------------------
// Wrap / WrapPrefix
// ---------------------------------------------------------------------------

func TestWrapError(t *testing.T) {
	e := func() error {
		return Wrap(fmt.Errorf("hi"), 1)
	}()

	if e.Error() != "hi" {
		t.Errorf("Constructor with a string failed")
	}

	if Wrap(fmt.Errorf("yo"), 0).Error() != "yo" {
		t.Errorf("Constructor with an error failed")
	}

	if Wrap(e, 0) != e {
		t.Errorf("Constructor with an Error failed")
	}

	if Wrap(nil, 0) != nil {
		t.Errorf("Constructor with nil failed")
	}
}

func TestWrap_hasLocation(t *testing.T) {
	err := Wrap(fmt.Errorf("test"), 0).(*Error)
	// Wrap does not eagerly resolve location; it falls back via StackFrames.
	file, line := err.Location()
	if file == "" || line == 0 {
		t.Errorf("Wrap should provide location (via StackFrames fallback), got file=%q line=%d", file, line)
	}
	if !strings.HasSuffix(file, "errors_test.go") {
		t.Errorf("Wrap location file should end with errors_test.go, got %q", file)
	}
}

func TestWrapPrefixError(t *testing.T) {
	e := func() error {
		return WrapPrefix(fmt.Errorf("hi"), "prefix", 1)
	}()

	if e.Error() != "prefix: hi" {
		t.Errorf("Constructor with a string failed")
	}

	if WrapPrefix(fmt.Errorf("yo"), "prefix", 0).Error() != "prefix: yo" {
		t.Errorf("Constructor with an error failed")
	}

	prefixed := WrapPrefix(e, "prefix", 0)
	prefixedErr := prefixed.(*Error)
	original := e.(*Error)

	if prefixedErr.Err != original || prefixed.Error() != "prefix: prefix: hi" {
		t.Errorf("Constructor with an Error failed: got Err=%v, Error=%q", prefixedErr.Err, prefixed.Error())
	}

	if original.Error() == prefixed.Error() {
		t.Errorf("WrapPrefix changed the original error")
	}

	if WrapPrefix(nil, "prefix", 0) != nil {
		t.Errorf("Constructor with nil failed")
	}

	locFile, _ := prefixedErr.Location()
	if !strings.HasSuffix(locFile, "errors_test.go") {
		t.Errorf("Location failed: got %q", locFile)
	}
}

func TestWrapPrefix_lightweight(t *testing.T) {
	// WrapPrefix intentionally does NOT capture a full stack — only location.
	// This keeps chained wrapping fast.
	err := WrapPrefix(fmt.Errorf("base"), "ctx", 0).(*Error)
	if len(err.Callers()) != 0 {
		t.Errorf("WrapPrefix should NOT capture full stack, got %d callers", len(err.Callers()))
	}
	// But it should have location.
	file, line := err.Location()
	if file == "" || line == 0 {
		t.Errorf("WrapPrefix should have location, got file=%q line=%d", file, line)
	}
}

// ---------------------------------------------------------------------------
// ErrUnsupported
// ---------------------------------------------------------------------------

func TestErrUnsupported(t *testing.T) {
	if ErrUnsupported == nil {
		t.Fatal("ErrUnsupported should not be nil")
	}
	err := fmt.Errorf("wrapped: %w", ErrUnsupported)
	if !Is(err, ErrUnsupported) {
		t.Errorf("should match ErrUnsupported through wrapping")
	}
}

// ---------------------------------------------------------------------------
// Join / Unwrap
// ---------------------------------------------------------------------------

func TestJoin(t *testing.T) {
	err1 := New("err1")
	err2 := New("err2")

	joined := Join(err1, err2)
	if joined == nil {
		t.Fatal("Join should not return nil for non-nil errors")
	}

	if !Is(joined, err1) {
		t.Errorf("joined error should match err1")
	}
	if !Is(joined, err2) {
		t.Errorf("joined error should match err2")
	}
}

func TestJoin_allNil(t *testing.T) {
	if Join(nil, nil) != nil {
		t.Errorf("Join of all nils should be nil")
	}
}

func TestUnwrap(t *testing.T) {
	inner := fmt.Errorf("inner")
	outer := Wrap(inner, 0)
	if Unwrap(outer) != inner {
		t.Errorf("Unwrap should return the inner error")
	}
}

// ---------------------------------------------------------------------------
// StackFrames thread safety
// ---------------------------------------------------------------------------

func TestStackFrames_concurrent(t *testing.T) {
	e := From(io.EOF)
	var wg [10]chan struct{}
	for i := range wg {
		wg[i] = make(chan struct{})
		go func(ch chan struct{}) {
			_ = e.StackFrames()
			close(ch)
		}(wg[i])
	}
	for _, ch := range wg {
		<-ch
	}
	// If we get here without a race detector complaint, we're good.
	if len(e.StackFrames()) == 0 {
		t.Errorf("expected non-empty stack frames")
	}
}

// ---------------------------------------------------------------------------
// FuncName / LocationFunc backward compat
// ---------------------------------------------------------------------------

func TestFuncName(t *testing.T) {
	e := From(io.EOF)
	fn := e.FuncName()
	if fn == "" {
		t.Errorf("FuncName should not be empty")
	}
	if fn != e.LocationFunc() {
		t.Errorf("FuncName and LocationFunc should return the same value")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func a() error {
	b(5)
	return nil
}

func b(i int) {
	c()
}

func c() {
	panic('a')
}

// compareStacks compares a stack created using this package (actual) with a
// reference stack created with the callers function (expected). The first
// entry is not compared since the actual and expected stacks cannot be
// created at the exact same program counter position, so the first entry
// will always differ somewhat.
func compareStacks(actual, expected []uintptr) error {
	if len(actual) != len(expected) {
		return stackCompareError("Stacks does not have equal length", actual, expected)
	}
	for i, pc := range actual {
		if i != 0 && pc != expected[i] {
			return stackCompareError(fmt.Sprintf("Stacks does not match entry %d (and maybe others)", i), actual, expected)
		}
	}
	return nil
}

func stackCompareError(msg string, actual, expected []uintptr) error {
	return fmt.Errorf("%s\nActual stack trace:\n%s\nExpected stack trace:\n%s", msg, readableStackTrace(actual), readableStackTrace(expected))
}

func callers() []uintptr {
	return callersSkip(1)
}

func callersSkip(skip int) []uintptr {
	callers := make([]uintptr, MaxStackDepth)
	length := runtime.Callers(skip+2, callers[:])
	return callers[:length]
}

func readableStackTrace(callers []uintptr) string {
	var result bytes.Buffer
	frames := callersToFrames(callers)
	for _, frame := range frames {
		result.WriteString(fmt.Sprintf("%s:%d (%#x)\n\t%s\n", frame.File, frame.Line, frame.PC, frame.Function))
	}
	return result.String()
}

func callersToFrames(callers []uintptr) []runtime.Frame {
	frames := make([]runtime.Frame, 0, len(callers))
	framesPtr := runtime.CallersFrames(callers)
	for {
		frame, more := framesPtr.Next()
		frames = append(frames, frame)
		if !more {
			return frames
		}
	}
}

type errorString string

func (e errorString) Error() string {
	return string(e)
}

type errorWithCustomIs struct {
	Key string
	Err error
}

func (ewci errorWithCustomIs) Error() string {
	return "[" + ewci.Key + "]: " + ewci.Err.Error()
}

func (ewci errorWithCustomIs) Is(target error) bool {
	matched, ok := target.(errorWithCustomIs)
	return ok && matched.Key == ewci.Key
}
