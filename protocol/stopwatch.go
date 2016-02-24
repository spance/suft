package suft

import (
	"fmt"
	"os"
	"time"
)

type watch struct {
	label string
	t1    time.Time
}

func StartWatch(s string) *watch {
	return &watch{
		label: s,
		t1:    time.Now(),
	}
}

func (w *watch) StopLoops(loop int, size int) {
	tu := time.Now().Sub(w.t1).Nanoseconds()
	timePerLoop := float64(tu) / float64(loop)
	thoughput := float64(loop*size) * 1e6 / float64(tu)
	tu_ms := float64(tu) / 1e6
	fmt.Fprintf(os.Stderr, "%s tu=%.2f ms tpl=%.0f ns thoughput=%.2f K/s\n", w.label, tu_ms, timePerLoop, thoughput)
}

var _kt = float64(1e9 / 1024)

func (w *watch) Stop(size int) {
	tu := time.Now().Sub(w.t1).Nanoseconds()
	thoughput := float64(size) * _kt / float64(tu)
	tu_ms := float64(tu) / 1e6
	fmt.Fprintf(os.Stderr, "%s tu=%.2f ms thoughput=%.2f K/s\n", w.label, tu_ms, thoughput)
}
