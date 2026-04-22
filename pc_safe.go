//go:build safe || !(amd64 || arm64)

package errors

import "runtime"

func getCallerPC() uintptr {
	var rpc [1]uintptr
	n := runtime.Callers(3, rpc[:])
	if n == 0 {
		return 0
	}
	return rpc[0]
}
