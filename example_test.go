package errors_test

import (
	"fmt"
	"io"
	"os"

	errors "github.com/go-errors/errors"
)

func ExampleNew() {
	err := errors.New("something failed")
	fmt.Println(err)
	// Output: something failed
}

func ExampleErrorf() {
	err := errors.Errorf("unexpected value: %d", 42)
	fmt.Println(err)
	// Output: unexpected value: 42
}

func ExampleWrap() {
	err := errors.Wrap(io.EOF, 0)
	fmt.Println(err)
	// Output: EOF
}

func ExampleWrapPrefix() {
	err := errors.WrapPrefix(io.EOF, "read config", 0)
	fmt.Println(err)
	// Output: read config: EOF
}

func ExampleIs() {
	err := errors.Wrap(io.EOF, 0)
	if errors.Is(err, io.EOF) {
		fmt.Println("matches io.EOF")
	}
	// Output: matches io.EOF
}

func ExampleAs() {
	pathErr := &os.PathError{Op: "open", Path: "/no/such/file", Err: os.ErrNotExist}
	wrapped := errors.Wrap(pathErr, 0)

	var target *os.PathError
	if errors.As(wrapped, &target) {
		fmt.Println("path:", target.Path)
	}
	// Output: path: /no/such/file
}

func ExampleJoin() {
	err1 := errors.New("first")
	err2 := errors.New("second")
	joined := errors.Join(err1, err2)
	fmt.Println(joined)
	// Output:
	// first
	// second
}

func ExampleFrom() {
	err := errors.From(io.EOF)
	fmt.Println(err.Error())
	// Output: EOF
}

func ExampleError_ErrorStack() {
	err := errors.From(io.EOF)
	// ErrorStack returns the type, message, and full stack trace.
	stack := err.ErrorStack()
	_ = stack // typically printed or logged
}

func ExampleError_TypeName() {
	err := errors.From(io.EOF)
	fmt.Println(err.TypeName())
	// Output: *errors.errorString
}
