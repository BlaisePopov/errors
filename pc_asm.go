//go:build (amd64 || arm64) && !safe

package errors

func getCallerPC() uintptr
