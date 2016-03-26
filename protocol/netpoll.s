#include "textflag.h"

TEXT ·net_pollSetDeadline(SB),NOSPLIT,$0
	JMP	net·runtime_pollSetDeadline(SB)

TEXT ·popcnt(SB),NOSPLIT,$0
	XORQ	AX, AX
	MOVQ	x+0(FP), SI
	BYTE	$0xf3; BYTE $0x48; BYTE $0x0f; BYTE $0xb8; BYTE $0xc6; //POPCNT (SI), AX
	MOVQ	AX, ret+8(FP)
	RET

