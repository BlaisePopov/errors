//go:build safe || !(amd64 || arm64)

package errors

import "runtime"

// getCallerPC returns the program counter of the caller using runtime.Callers.
// This is the safe, portable fallback. It is selected automatically on
// non-amd64/arm64 platforms, or when building with -tags safe.
func getCallerPC() uintptr {
	var rpc [1]uintptr
	n := runtime.Callers(3, rpc[:])
	if n == 0 {
		return 0
	}
	return rpc[0]
}
