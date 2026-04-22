#include "textflag.h"

// func getCallerPC() uintptr
TEXT ·getCallerPC(SB),NOSPLIT|NOFRAME,$0-8
	MOVD 8(R29), R20
	MOVD R20, ret+0(FP)
	RET
