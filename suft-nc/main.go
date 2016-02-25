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
	var laddr, raddr string
	var serv bool
	var p suft.Params
	flag.StringVar(&laddr, "l", ":9090", "local")
	flag.StringVar(&raddr, "r", ":9090", "remote")
	flag.BoolVar(&serv, "s", false, "is server")
	flag.BoolVar(&p.FastRetransmitEnabled, "fr", false, "enableFastRetransmit")
	flag.BoolVar(&p.SuperRetransmit, "sr", false, "superRetransmit")
	flag.Int64Var(&p.Bandwidth, "b", 2, "bandwidth in mbps")
	flag.IntVar(&p.Debug, "debug", 0, "debug")
	flag.BoolVar(&p.Stacktrace, "stacktrace", false, "stacktrace")
	flag.Parse()

	if !serv && raddr == ":9090" {
		log.Fatalln("missing -r")
	}
	suft.SetParams(&p)

	e, err := suft.NewEndpoint(laddr, serv)
	checkErr(err)
	log.Println("start", e.Addr())
	var conn *suft.Conn
	if !serv { // client
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
	case <-time.After(time.Millisecond * 200):
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
