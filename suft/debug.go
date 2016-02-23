package suft

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
)

type Params struct {
	FastRetransmitEnabled bool
	SuperRetransmit       bool
	Bandwidth             int64
	Debug                 int
	EnablePprof           bool
	Stacktrace            bool
}

func SetParams(p *Params) {
	fastRetransmitEnabled = p.FastRetransmitEnabled
	superRetransmit = p.SuperRetransmit
	debug = p.Debug
	enable_pprof = p.EnablePprof
	stacktrace = p.Stacktrace
	if p.Bandwidth <= 0 || p.Bandwidth > 100 {
		log.Fatalln("bw->(0,100]")
	}
	bandwidth = p.Bandwidth << 20 // mbps to bps
	if enable_pprof {
		f, err := os.Create("suft.pprof")
		if err != nil {
			log.Fatalln(err)
		}
		pprof.StartCPUProfile(f)
	}
}

var enable_pprof bool
var stacktrace bool

func (c *Conn) PrintState() {
	log.Printf("inQ=%d inQReady=%d outQ=%d", c.inQ.size(), len(c.inQReady), c.outQ.size())
	log.Printf("inMaxCtnSeq=%d lastAck=%d lastReadSeq=%d", c.inMaxCtnSeq, c.lastAck, c.lastReadSeq)
	if c.inPkCnt > 0 {
		log.Printf("rxpc=%d dups=%d %%d=%f%%", c.inPkCnt, c.inDupCnt, 100*float32(c.inDupCnt)/float32(c.inPkCnt))
	}
	if c.outPkCnt > 0 {
		log.Printf("txpc=%d dups=%d %%d=%f%%", c.outPkCnt, c.outDupCnt, 100*float32(c.outDupCnt)/float32(c.outPkCnt))
	}
	if fastRetransmitEnabled {
		log.Printf("FastRetransmit=%d", c.fRCnt)
	}
	if stacktrace {
		var buf = make([]byte, 6400)
		for i := 0; i < 3; i++ {
			n := runtime.Stack(buf, true)
			if n >= len(buf) {
				buf = make([]byte, len(buf)<<1)
			} else {
				buf = buf[:n]
				break
			}
		}
		fmt.Println(string(buf))
	}
	if enable_pprof {
		pprof.StopCPUProfile()
	}
}

func (c *Conn) internal_state() {
	ev := make(chan os.Signal, 10)
	signal.Notify(ev, syscall.Signal(12), syscall.SIGINT)
	for v := range ev {
		c.PrintState()
		if v == syscall.SIGINT {
			os.Exit(1)
		}
	}
}

func printBits(b uint64, j, s, d uint32) {
	fmt.Printf("bits=%064b j=%d seq=%d dis=%d\n", b, j, s, d)
}

func dumpb(label string, buf []byte) {
	log.Println(label, "\n", hex.Dump(buf))
}

func dumpQ(s string, q *linkedMap) {
	var seqs = make([]uint32, 0, 20)
	n := q.head
	for i, m := int32(0), q.size(); i < m && n != nil; i++ {
		seqs = append(seqs, n.seq)
		n = n.next
		if len(seqs) == 20 {
			log.Printf("%s: Q=%d", s, seqs)
			seqs = seqs[:0]
		}
	}
	if len(seqs) > 0 {
		log.Printf("%s: Q=%d", s, seqs)
	}
}
