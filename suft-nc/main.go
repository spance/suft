package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/spance/suft/protocol"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Flags() | log.Lmicroseconds)
}

var wto int64
var waiting = make(chan *eofStatus, 2)

func main() {
	var raddr string
	var p suft.Params
	flag.StringVar(&p.LocalAddr, "l", "", "local")
	flag.StringVar(&raddr, "r", ":9090", "remote")
	flag.BoolVar(&p.IsServ, "s", false, "is server")
	flag.BoolVar(&p.FastRetransmit, "fr", false, "enableFastRetransmit")
	flag.Int64Var(&p.Bandwidth, "b", 2, "bandwidth in mbps")
	flag.IntVar(&p.Debug, "debug", 0, "debug")
	flag.BoolVar(&p.EnablePprof, "pprof", false, "pprof")
	flag.BoolVar(&p.Stacktrace, "stacktrace", false, "stacktrace")
	flag.BoolVar(&p.FlatTraffic, "ft", false, "FlatTraffic")
	flag.Int64Var(&wto, "wto", 0, "timeout of waiting for both eof")
	flag.Parse()

	if !p.IsServ && raddr == ":9090" {
		log.Fatalln("missing -r")
	}

	e, err := suft.NewEndpoint(&p)
	checkErr(err)
	defer e.Close()

	log.Println("start", e.Addr())
	var conn *suft.Conn
	if !p.IsServ { // client
		conn, err = e.Dial(raddr)
		checkErr(err)
		log.Println("connected to", conn.RemoteAddr())
		go writeOut(conn)
		go readIn(conn)
	} else {
		conn = e.Listen()
		log.Println("connected from", conn.RemoteAddr())
		go writeOut(conn)
		go readIn(conn)
	}

	var eof1, eof2 *eofStatus
	eof1 = <-waiting
	log.Println(eof1.msg)

	if wto > 0 {
		log.Printf("the countdown to %c has started", "RW"[(eof1.channel+1)%2])
		select {
		case eof2 = <-waiting:
		case <-time.After(time.Duration(wto * 1e9)):
			conn.Close()
		}
	} else {
		eof2 = <-waiting
	}

	if eof2 != nil {
		log.Println(eof2.msg)
	}
	conn.PrintState()
}

func readIn(c *suft.Conn) {
	var (
		n          int64
		err1, err2 error
	)
	wa := suft.StartWatch("R")
	n, err1 = io.Copy(c, os.Stdin)
	wa.Stop(int(n))

	if wto <= 0 {
		err2 = c.Close()
	}
	waiting <- &eofStatus{
		channel: 0,
		msg:     fmt.Sprint("R done", err1, err2),
	}
}

func writeOut(c *suft.Conn) {
	var (
		n          int64
		err1, err2 error
	)
	wa := suft.StartWatch("W")
	n, err1 = io.Copy(os.Stdout, c)
	os.Stdout.Sync()
	wa.Stop(int(n))

	if wto <= 0 {
		err2 = c.Close()
	}
	waiting <- &eofStatus{
		channel: 1,
		msg:     fmt.Sprint("W done", err1, err2),
	}
}

func checkErr(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}

type eofStatus struct {
	channel int
	msg     string
}
