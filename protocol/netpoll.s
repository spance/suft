#include "textflag.h"

TEXT ·net_pollSetDeadline(SB),NOSPLIT,$0
	JMP	net·runtime_pollSetDeadline(SB)
