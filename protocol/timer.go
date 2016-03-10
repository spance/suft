package suft

const Millisecond = 1e6

type iTimer struct {
	C chan byte
	r runtimeTimer
}

type runtimeTimer struct {
	i      int
	when   int64
	period int64
	f      func(c interface{}, seq uintptr) // NOTE: must not be closure
	arg    interface{}
	seq    uintptr
}

func startTimer(*runtimeTimer)
func stopTimer(*runtimeTimer) bool
func runtimeNano() int64
func currentTime() (sec int64, nsec int32)

func Now() int64 {
	s, n := currentTime()
	return s*1e3 + int64(n/1e6)
}

func NowNS() int64 {
	s, n := currentTime()
	return s*1e9 + int64(n)
}

func NewTimer(d int64) *iTimer {
	t := iTimer{
		C: make(chan byte, 1),
		r: runtimeTimer{
			i:    -1,
			when: d*Millisecond + runtimeNano(),
			f:    timerCallback,
		},
	}
	t.r.arg = t.C
	return &t
}

func timerCallback(c interface{}, seq uintptr) {
	select {
	case c.(chan byte) <- 1:
	default:
	}
}

func (t *iTimer) Stop() {
	stopTimer(&t.r)
}

func (t *iTimer) Reset(d int64) {
	stopTimer(&t.r)
	select {
	case <-t.C:
	default:
	}
	t.r.when = d*Millisecond + runtimeNano()
	startTimer(&t.r)
}

func (t *iTimer) TryActive(d int64) {
	if t.r.i < 0 {
		select {
		case <-t.C:
		default:
		}
		t.r.when = d*Millisecond + runtimeNano()
		startTimer(&t.r)
	}
}

func NewTimerChan(d int64) <-chan byte {
	t := iTimer{
		C: make(chan byte, 1),
		r: runtimeTimer{
			i:    -1,
			when: d*Millisecond + runtimeNano(),
			f:    timerCallback,
		},
	}
	t.r.arg = t.C
	startTimer(&t.r)
	return t.C
}
