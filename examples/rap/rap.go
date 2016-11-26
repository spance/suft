package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/spance/suft/protocol"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Flags() | log.Lmicroseconds)
}

func main() {
	var raddr string
	var p suft.Params

	flag.StringVar(&p.LocalAddr, "l", "", "local")
	flag.StringVar(&raddr, "r", "", "AtServer => backend_tcp_peer, AtClient => remote_suft_peer")
	flag.BoolVar(&p.IsServ, "s", false, "is server")
	flag.BoolVar(&p.FastRetransmit, "fr", true, "FastRetransmit")
	flag.BoolVar(&p.FlatTraffic, "ft", true, "FlatTraffic")
	flag.Int64Var(&p.Bandwidth, "b", 4, "bandwidth in mbps")

	flag.IntVar(&p.Debug, "debug", 0, "debug")
	flag.BoolVar(&p.EnablePprof, "pprof", false, "pprof")
	flag.BoolVar(&p.Stacktrace, "stacktrace", false, "stacktrace")
	flag.Parse()

	if !p.IsServ && raddr == "" {
		log.Fatalln("missing -r")
	}

	e, err := suft.NewEndpoint(&p)
	checkErr(err)
	defer e.Close()
	log.Println("start", e.Addr())

	var suConn *suft.Conn
	if !p.IsServ { // client
		ln, err := net.Listen("tcp", p.LocalAddr)
		checkErr(err)
		defer ln.Close()

		for {
			local, err := ln.Accept()
			checkErr(err)
			suConn, err = e.Dial(raddr)
			if checkWarn(err) {
				log.Printf("connected to %s for %s", suConn.RemoteAddr(), local.RemoteAddr())
				go duplexPipe(suConn, local)
			} else {
				safeClose(suConn)
			}
		}
	} else {
		for {
			suConn = e.Listen()
			backend, err := net.Dial("tcp", raddr)
			if checkWarn(err) {
				log.Printf("connected from %s to %s", suConn.RemoteAddr(), backend.RemoteAddr())
				go duplexPipe(suConn, backend)
			} else {
				safeClose(backend)
			}
		}
	}
}

func checkErr(e error) {
	if e != nil {
		log.Panicln(e)
	}
}

func checkWarn(e error) bool {
	if e != nil {
		log.Println(e)
		return false
	}
	return true
}

func safeClose(c io.Closer) {
	if c != nil {
		c.Close()
	}
}

func duplexPipe(s *suft.Conn, t net.Conn) {
	const BUF_SIZE = 1 << 20
	t.(*net.TCPConn).SetNoDelay(true)
	defer t.Close()
	defer s.Close()

	var wait = make(chan byte, 2)
	go func() {
		buf := make([]byte, BUF_SIZE)
		io.CopyBuffer(s, t, buf)
		wait <- 1
	}()
	go func() {
		buf := make([]byte, BUF_SIZE)
		io.CopyBuffer(t, s, buf)
		wait <- 1
	}()
	// one channel closed
	<-wait
	select {
	case <-wait: // second
	case <-time.After(10e9):
	}
}
