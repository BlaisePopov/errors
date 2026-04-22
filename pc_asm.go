//go:build (amd64 || arm64) && !safe

package errors

// getCallerPC returns the program counter of the caller.
//
// SECURITY NOTE: The assembly implementations in pc_asm_amd64.s and
// pc_asm_arm64.s read the return address from a fixed stack offset (BP+8
// on amd64, R29+8 on arm64). This relies on the current Go calling convention
// and is not guaranteed by the Go spec. A future Go version could break this.
//
// To use the safe (slower) fallback based on runtime.Callers instead, build
// with the "safe" tag:
//
//	go build -tags safe
//
// The safe version adds ~20-40ns overhead per error creation call, which is
// negligible compared to the total error creation cost of ~400-900ns.
func getCallerPC() uintptr
