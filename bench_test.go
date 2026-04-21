package errors

import (
	"fmt"
	"io"
	"testing"
)

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = New("benchmark error")
	}
}

func BenchmarkWrap(b *testing.B) {
	b.ReportAllocs()
	err := fmt.Errorf("base")
	for i := 0; i < b.N; i++ {
		_ = Wrap(err, 0)
	}
}

func BenchmarkWrapPrefix(b *testing.B) {
	b.ReportAllocs()
	err := fmt.Errorf("base")
	for i := 0; i < b.N; i++ {
		_ = WrapPrefix(err, "ctx", 0)
	}
}

func BenchmarkError_String(b *testing.B) {
	b.ReportAllocs()
	err := Wrap(fmt.Errorf("something failed"), 0)
	for i := 0; i < b.N; i++ {
		_ = err.Error()
	}
}

func BenchmarkStackFrames(b *testing.B) {
	b.ReportAllocs()
	err := Wrap(io.EOF, 0)
	_ = err.(*Error).StackFrames()
	for i := 0; i < b.N; i++ {
		_ = err.(*Error).StackFrames()
	}
}

func BenchmarkStack(b *testing.B) {
	b.ReportAllocs()
	err := Wrap(io.EOF, 0)
	for i := 0; i < b.N; i++ {
		_ = err.(*Error).Stack()
	}
}

func BenchmarkErrorStack(b *testing.B) {
	b.ReportAllocs()
	err := Wrap(io.EOF, 0)
	for i := 0; i < b.N; i++ {
		_ = err.(*Error).ErrorStack()
	}
}

func BenchmarkFrom(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = From(io.EOF)
	}
}

func BenchmarkWrapPrefix_chained(b *testing.B) {
	b.ReportAllocs()
	base := fmt.Errorf("base")
	for i := 0; i < b.N; i++ {
		e1 := WrapPrefix(base, "layer1", 0)
		e2 := WrapPrefix(e1, "layer2", 0)
		_ = WrapPrefix(e2, "layer3", 0)
	}
}
