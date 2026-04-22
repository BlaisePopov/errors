#include "textflag.h"

// func getCallerPC() uintptr
TEXT ·getCallerPC(SB),NOSPLIT|NOFRAME,$0-8
	MOVQ 8(BP), AX
	MOVQ AX, ret+0(FP)
	RET
