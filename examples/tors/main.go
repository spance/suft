package main

import (
	"encoding/binary"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/spance/suft/protocol"
)

var (
	transMode int
	endpoint  *suft.Endpoint
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetOutput(os.Stderr)
}

func main() {
	var lAddr, rAddr, trans string
	var bw int64
	flag.StringVar(&lAddr, "l", "", "listen addr for server")
	flag.StringVar(&rAddr, "r", "", "remote addr for client")
	flag.StringVar(&trans, "t", "suft", "transport layer")
	flag.Int64Var(&bw, "b", 4, "bandwidth in mbps")
	flag.Parse()

	if (lAddr != "" && rAddr != "") || (lAddr == "" && rAddr == "") {
		log.Fatalln("-l for server and -r for client")
	}
	var isServ bool
	var err error
	isServ = !(rAddr != "" && flag.NArg() >= 2)

	switch trans {
	case "suft":
		transMode = 'S'
		endpoint, err = suft.NewEndpoint(&suft.Params{
			LocalAddr:      lAddr,
			Bandwidth:      bw,
			FastRetransmit: true,
			FlatTraffic:    true,
			IsServ:         isServ,
		})
		abortIf(err)

	case "tcp":
		transMode = 'T'

	default:
		log.Fatalln("Unknown -t", trans)
	}

	if isServ {
		listen(lAddr)
	} else {
		connect(rAddr, flag.Args())
	}

}

func netListen(addr string) net.Listener {
	if transMode == 'T' {
		ln, err := net.Listen("tcp", addr)
		abortIf(err)
		return ln
	} else {
		return endpoint
	}
}

func netDial(addr string) net.Conn {
	var conn net.Conn
	var err error
	if transMode == 'T' {
		conn, err = net.Dial("tcp", addr)
	} else {
		conn, err = endpoint.Dial(addr)
	}
	abortIf(err)
	return conn
}

func listen(addr string) {
	ln := netListen(addr)
	defer ln.Close()
	log.Println("listen at", ln.Addr())

	for {
		conn, err := ln.Accept()
		if err == nil {
			log.Println("Client from", conn.RemoteAddr())
			go sessionHandle(conn)
		} else if isTemporaryError(err) {
			continue
		} else {
			break
		}
	}
}

func sessionHandle(conn net.Conn) {
	defer conn.Close()

	cmdline := readInitialHeader(conn)
	if !(len(cmdline) > 1 && cmdline[0] == "rsync") {
		return
	}
	log.Printf("RUN %s", cmdline)

	cmd := exec.Command(cmdline[0], cmdline[1:]...)
	stdin, err := cmd.StdinPipe()
	abortIf(err)
	stdout, err := cmd.StdoutPipe()
	abortIf(err)
	stderr, err := cmd.StderrPipe()
	abortIf(err)
	abortIf(cmd.Start())

	go func() {
		// conn -> cmd.stdin
		n, e := io.Copy(stdin, conn)
		e = consumeClosedError(e)
		log.Println("net/Rx", n, e)
	}()
	go func() {
		// cmd.stdout -> conn
		n, e := io.Copy(conn, stdout)
		log.Println("net/Tx", n, e)
	}()
	// cmd.stderr -> current.stderr
	go io.Copy(os.Stderr, stderr)

	err = cmd.Wait()
	conn.Close()
	log.Println("finished", err)
}

func connect(addr string, args []string) {
	conn := netDial(addr)
	defer conn.Close()
	writeInitialHeader(conn, args[1:])

	var done = make(chan int, 1)
	var stdin = getStdin()

	go func() {
		// stdin -> conn
		n, e := io.Copy(conn, stdin)
		log.Println("net/Tx", n, e)
		done <- 1
	}()
	go func() {
		// conn -> stdout
		n, e := io.Copy(getStdout(), conn)
		log.Println("net/Rx", n, e)
		done <- 2
	}()
	<-done
}

func isTemporaryError(err error) bool {
	if err != nil {
		if ne, y := err.(net.Error); y {
			return ne.Temporary()
		}
	}
	return false
}

func abortIf(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func readInitialHeader(rd net.Conn) []string {
	var l4 = make([]byte, 4)
	_, err := io.ReadFull(rd, l4)
	abortIf(err)
	nlen := binary.BigEndian.Uint32(l4)
	var cmd = make([]byte, nlen)
	_, err = io.ReadFull(rd, cmd)
	abortIf(err)
	return strings.Split(string(cmd), "\x00")
}

func writeInitialHeader(rd net.Conn, args []string) {
	var cmdline = strings.Join(args, "\x00")
	var l4 = make([]byte, 4)
	binary.BigEndian.PutUint32(l4, uint32(len(cmdline)))
	_, err := rd.Write(l4)
	abortIf(err)
	_, err = rd.Write([]byte(cmdline))
	abortIf(err)
}

func consumeClosedError(e error) error {
	if e != nil && strings.Contains(e.Error(), "closed") {
		return nil
	} else {
		return e
	}
}
