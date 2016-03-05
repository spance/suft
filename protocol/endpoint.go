package suft

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cloudflare/golibs/bytepool"
)

const (
	_SO_BUF_SIZE = 8 << 20
)

var (
	bpool bytepool.BytePool
)

type Params struct {
	LocalAddr       string
	Bandwidth       int64
	Debug           int
	IsServ          bool
	FastRetransmit  bool
	SuperRetransmit bool
	EnablePprof     bool
	Stacktrace      bool
}

type connId struct {
	lid uint32
	rid uint32
}

type Endpoint struct {
	udpconn    *net.UDPConn
	state      int32
	idSeq      uint32
	isServ     bool
	listenChan chan *Conn
	registry   map[uint32]*Conn
	mlock      sync.RWMutex
	timeout    *iTimer
	params     Params
}

func (c *connId) setRid(b []byte) {
	c.rid = binary.BigEndian.Uint32(b[MAGIC_SIZE+6:])
}

func init() {
	bpool.Init(0, 2000)
}

func NewEndpoint(p *Params) (*Endpoint, error) {
	set_debug_params(p)
	if p.Bandwidth <= 0 || p.Bandwidth > 100 {
		return nil, fmt.Errorf("bw->(0,100]")
	}
	conn, err := net.ListenPacket("udp", p.LocalAddr)
	if err != nil {
		return nil, err
	}
	e := &Endpoint{
		udpconn:    conn.(*net.UDPConn),
		idSeq:      1,
		isServ:     p.IsServ,
		listenChan: make(chan *Conn, 1),
		registry:   make(map[uint32]*Conn),
		timeout:    NewTimer(0),
		params:     *p,
	}
	if e.isServ {
		e.state = S_EST0
	} else {
		e.state = S_EST1
		e.idSeq = 0xff0
	}
	e.params.Bandwidth = p.Bandwidth << 20 // mbps to bps
	e.udpconn.SetReadBuffer(_SO_BUF_SIZE)
	e.udpconn.SetWriteBuffer(_SO_BUF_SIZE)
	go e.internal_listen()
	return e, nil
}

func (e *Endpoint) internal_listen() {
	const rtmo = 60 * 1e9
	var pdCtx = getPollCtx(e.udpconn)
	for {
		//var buf = make([]byte, 1600)
		var buf = bpool.Get(1600)
		net_pollSetDeadline(pdCtx, rtmo, 'r')
		n, addr, err := e.udpconn.ReadFromUDP(buf)
		if err == nil && n >= AH_SIZE {
			var id = e.getConnId(buf)
			buf = buf[:n]

			switch id.lid {
			case 0: // new connection
				if e.isServ {
					go e.acceptNewConn(id, addr, buf)
				} else {
					dumpb("drop", buf)
				}

			case _INVALID_SEQ:
				dumpb("drop invalid", buf)

			default: // old connection
				e.mlock.RLock()
				conn := e.registry[id.lid]
				e.mlock.RUnlock()
				if conn != nil {
					e.dispatch(conn, buf)
				} else {
					dumpb("drop null", buf)
				}
			}

		} else {
			// idle process
			if nerr, y := err.(net.Error); y && nerr.Timeout() {
				bpool.Drain()
				continue
			}
			// other errors
			fmt.Println("err", err)
			if atomic.LoadInt32(&e.state) == S_FIN {
				return
			}
		}
	}
}

func (e *Endpoint) Dial(addr string) (*Conn, error) {
	rAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}
	e.mlock.Lock()
	e.idSeq++
	id := connId{e.idSeq, 0}
	conn := NewConn(e, rAddr, id)
	e.registry[id.lid] = conn
	e.mlock.Unlock()
	if atomic.LoadInt32(&e.state) == S_FIN {
		return nil, io.EOF
	}
	err = conn.initConnection(nil)
	return conn, err
}

func (e *Endpoint) acceptNewConn(id connId, addr *net.UDPAddr, buf []byte) {
	e.mlock.Lock()
	e.idSeq++
	id.lid = e.idSeq
	conn := NewConn(e, addr, id)
	e.registry[id.lid] = conn
	e.mlock.Unlock()
	err := conn.initConnection(buf)
	if err == nil {
		select {
		case e.listenChan <- conn:
		case <-time.After(_10ms):
			fmt.Println("Warn no listening")
		}
	} else {
		e.removeConn(id)
		fmt.Println("Error init_connection", err)
	}
}

func (e *Endpoint) removeConn(id connId) {
	e.mlock.Lock()
	delete(e.registry, id.lid)
	e.mlock.Unlock()
}

// net.Listener
func (e *Endpoint) Close() error {
	state := atomic.LoadInt32(&e.state)
	if state > 0 && atomic.CompareAndSwapInt32(&e.state, state, S_FIN) {
		err := e.udpconn.Close()
		e.registry = make(map[uint32]*Conn)
		select { // release listeners
		case e.listenChan <- nil:
		default:
		}
		return err
	}
	return nil
}

// net.Listener
func (e *Endpoint) Addr() net.Addr {
	return e.udpconn.LocalAddr()
}

// net.Listener
func (e *Endpoint) Accept() (net.Conn, error) {
	if atomic.LoadInt32(&e.state) == S_EST0 {
		return <-e.listenChan, nil
	} else {
		return nil, io.EOF
	}
}

func (e *Endpoint) Listen() *Conn {
	if atomic.LoadInt32(&e.state) == S_EST0 {
		return <-e.listenChan
	} else {
		return nil
	}
}

// tmo in MS
func (e *Endpoint) ListenTimeout(tmo int64) *Conn {
	if tmo <= 0 {
		return e.Listen()
	}
	if atomic.LoadInt32(&e.state) == S_EST0 {
		select {
		case c := <-e.listenChan:
			return c
		case <-NewTimerChan(tmo):
		}
	}
	return nil
}

func (e *Endpoint) getConnId(buf []byte) connId {
	// TODO determine magic header
	// magic := binary.BigEndian.Uint64(buf)
	id := binary.BigEndian.Uint64(buf[MAGIC_SIZE+2:])
	return connId{uint32(id >> 32), uint32(id)}
}

func (e *Endpoint) dispatch(c *Conn, buf []byte) {
	e.timeout.Reset(30)
	select {
	case c.evRecv <- buf:
	case <-e.timeout.C:
		fmt.Println("dispatch failed")
	}
}
