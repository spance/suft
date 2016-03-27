#include "textflag.h"

TEXT ·currentTime(SB),NOSPLIT,$0
	JMP	time·now(SB)

TEXT ·startTimer(SB),NOSPLIT,$0
	JMP	time·startTimer(SB)

TEXT ·stopTimer(SB),NOSPLIT,$0
	JMP time·stopTimer(SB)

TEXT ·runtimeNano(SB),NOSPLIT,$0
	JMP time·runtimeNano(SB)

TEXT ·net_pollSetDeadline(SB),NOSPLIT,$0
	JMP	net·runtime_pollSetDeadline(SB)

