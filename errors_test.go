package errors

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestIs(t *testing.T) {
	t.Run("nil err", func(t *testing.T) {
		if Is(nil, io.EOF) {
			t.Errorf("nil should not match io.EOF")
		}
	})

	t.Run("identical sentinels", func(t *testing.T) {
		if !Is(io.EOF, io.EOF) {
			t.Errorf("io.EOF should match io.EOF")
		}
	})

	t.Run("wrapped matches inner", func(t *testing.T) {
		if !Is(From(io.EOF), io.EOF) {
			t.Errorf("From(io.EOF) should match io.EOF via Unwrap")
		}
	})

	t.Run("different errors", func(t *testing.T) {
		if Is(io.EOF, fmt.Errorf("io.EOF")) {
			t.Errorf("io.EOF should not match fmt.Errorf(\"io.EOF\")")
		}
	})

	t.Run("custom Is method", func(t *testing.T) {
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
			name   string
			err    error
			target error
			want   bool
		}{
			{"direct match", custErr, shouldMatch, true},
			{"direct no match", custErr, shouldNotMatch, false},
			{"wrap target match", custErr, Wrap(shouldMatch, 0), false},
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
	})
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

func TestStack_traceFromChain(t *testing.T) {
	inner := fmt.Errorf("base")
	wrapped := WrapPrefix(WrapPrefix(inner, "ctx1", 0), "ctx2", 0)
	e := wrapped.(*Error)

	frames := e.StackFrames()
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames from chain, got %d", len(frames))
	}

	for _, f := range frames {
		if f.File == "" || f.LineNumber == 0 {
			t.Errorf("frame should have file/line, got File=%q Line=%d", f.File, f.LineNumber)
		}
		if !strings.HasSuffix(f.File, "errors_test.go") {
			t.Errorf("frame file should end with errors_test.go, got %q", f.File)
		}
	}

	stack := string(e.Stack())
	if !strings.Contains(stack, "errors_test.go:") {
		t.Errorf("Stack trace does not contain file name: 'errors_test.go:'")
		t.Error(stack)
	}
}

func TestNew(t *testing.T) {
	err := New("foo")

	if err.Error() != "foo" {
		t.Errorf("Wrong message")
	}

	fromErr := From(fmt.Errorf("foo"))

	if fromErr.Error() != "foo" {
		t.Errorf("Wrong message")
	}

	e := err.(*Error)
	if e.pc == 0 {
		t.Errorf("New should capture a non-zero PC")
	}

	frames := e.StackFrames()
	if len(frames) != 1 {
		t.Errorf("expected 1 frame from New, got %d", len(frames))
	}
	if len(frames) > 0 {
		if !strings.HasSuffix(frames[0].File, "errors_test.go") {
			t.Errorf("frame file should end with errors_test.go, got %q", frames[0].File)
		}
	}

	if e.ErrorStack() != e.TypeName()+" "+e.Error()+"\n"+string(e.Stack()) {
		t.Errorf("ErrorStack is in the wrong format")
	}
}

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
	file, line := err.Location()
	if file == "" || line == 0 {
		t.Errorf("Wrap should provide location, got file=%q line=%d", file, line)
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

func TestWrapPrefix_hasPC(t *testing.T) {
	err := WrapPrefix(fmt.Errorf("base"), "ctx", 0).(*Error)
	if err.pc == 0 {
		t.Errorf("WrapPrefix should capture a PC")
	}
	file, line := err.Location()
	if file == "" || line == 0 {
		t.Errorf("WrapPrefix should have location, got file=%q line=%d", file, line)
	}
}

func TestErrUnsupported(t *testing.T) {
	if ErrUnsupported == nil {
		t.Fatal("ErrUnsupported should not be nil")
	}
	err := fmt.Errorf("wrapped: %w", ErrUnsupported)
	if !Is(err, ErrUnsupported) {
		t.Errorf("should match ErrUnsupported through wrapping")
	}
}

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
	if len(e.StackFrames()) == 0 {
		t.Errorf("expected non-empty stack frames")
	}
}

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

func TestError_caching(t *testing.T) {
	e := WrapPrefix(fmt.Errorf("base"), "ctx", 0).(*Error)
	msg1 := e.Error()
	msg2 := e.Error()
	if msg1 != msg2 {
		t.Errorf("cached Error() should return same value")
	}
	if msg1 != "ctx: base" {
		t.Errorf("expected 'ctx: base', got %q", msg1)
	}
}

func TestCallers(t *testing.T) {
	e1 := WrapPrefix(WrapPrefix(fmt.Errorf("base"), "ctx1", 0), "ctx2", 0).(*Error)
	pcs := e1.Callers()
	if len(pcs) != 2 {
		t.Errorf("expected 2 callers from chain, got %d", len(pcs))
	}
	for _, pc := range pcs {
		if pc == 0 {
			t.Errorf("caller PC should not be zero")
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

func (e errorWithCustomIs) Error() string {
	return "[" + e.Key + "]: " + e.Err.Error()
}

func (e errorWithCustomIs) Is(target error) bool {
	matched, ok := target.(errorWithCustomIs)
	return ok && matched.Key == e.Key
}
