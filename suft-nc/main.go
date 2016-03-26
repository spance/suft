package main

import (
	"flag"
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

var waiting = make(chan byte, 2)

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
	flag.Parse()

	if !p.IsServ && raddr == ":9090" {
		log.Fatalln("missing -r")
	}

	e, err := suft.NewEndpoint(&p)
	checkErr(err)
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
	<-waiting
	conn.PrintState()
	select {
	case <-waiting:
	case <-time.After(time.Second):
	}
}

func readIn(c *suft.Conn) {
	wa := suft.StartWatch("R")
	n, err := io.Copy(c, os.Stdin)
	wa.Stop(int(n))
	log.Println("R done", err, c.Close())
	waiting <- 1
}

func writeOut(c *suft.Conn) {
	wa := suft.StartWatch("W")
	n, err := io.Copy(os.Stdout, c)
	os.Stdout.Sync()
	wa.Stop(int(n))
	log.Println("W done", err, c.Close())
	waiting <- 1
}

func checkErr(e error) {
	if e != nil {
		log.Fatalln(e)
	}
}
