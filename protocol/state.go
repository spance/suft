package suft

import (
	"errors"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	_10ms  = time.Millisecond * 10
	_100ms = time.Millisecond * 100
)

const (
	_FIN_ACK_SEQ uint32 = 0xffFF0000
	_INVALID_SEQ uint32 = 0xffFFffFF
)

var (
	ErrIOTimeout        error = &TimeoutError{}
	ErrUnknown                = errors.New("Unknown error")
	ErrInexplicableData       = errors.New("Inexplicable data")
	ErrTooManyAttempts        = errors.New("Too many attempts to connect")
)

type TimeoutError struct{}

func (e *TimeoutError) Error() string   { return "i/o timeout" }
func (e *TimeoutError) Timeout() bool   { return true }
func (e *TimeoutError) Temporary() bool { return true }

type Conn struct {
	sock   *net.UDPConn
	dest   *net.UDPAddr
	edp    *endpoint
	connId connId // 8 bytes
	// events
	evRecv  chan []byte
	evRead  chan byte
	evSend  chan byte
	evSWnd  chan byte
	evAck   chan byte
	evClose chan byte
	// protocol state
	inlock      sync.Mutex
	outlock     sync.Mutex
	state       int32
	mySeq       uint32
	swnd        int32
	cwnd        int32
	missed      int32
	outPending  int32
	lastAck     uint32
	lastAckTime int64
	lastShrink  int64
	ato         int64
	rto         int64
	rtt         int64
	srtt        int64
	mdev        int64
	rtmo        int64
	wtmo        int64
	// queue
	outQ        *linkedMap
	inQ         *linkedMap
	inQReady    []byte
	inQDirty    bool
	inMaxCtnSeq uint32
	lastReadSeq uint32 // last user read seq
	// statistics
	inPkCnt   int
	inDupCnt  int
	outPkCnt  int
	outDupCnt int
	fRCnt     int
}

func NewConn(e *endpoint, dest *net.UDPAddr, id connId) *Conn {
	c := &Conn{
		sock:    e.udpconn,
		dest:    dest,
		edp:     e,
		connId:  id,
		evRecv:  make(chan []byte, 32),
		evRead:  make(chan byte, 1),
		evSWnd:  make(chan byte, 2),
		evSend:  make(chan byte, 4),
		evAck:   make(chan byte, 1),
		evClose: make(chan byte, 2),
		outQ:    NewLinkedMap(_QModeOut),
		inQ:     NewLinkedMap(_QModeIn),
	}
	return c
}

func (c *Conn) initConnection(buf []byte) (err error) {
	if buf == nil {
		err = c.initDialing()
	} else { //server
		err = c.acceptConnection(buf[TH_SIZE:])
	}
	if err != nil {
		return
	}
	if c.state == S_EST1 {
		c.lastReadSeq = c.lastAck
		c.inMaxCtnSeq = c.lastAck
		c.rtt = maxI64(c.rtt, MIN_RTT)
		c.mdev = c.rtt << 1
		c.srtt = c.rtt << 3
		c.rto = maxI64(c.rtt*3, MIN_RTO)
		c.ato = maxI64(c.rtt>>4, MIN_ATO)
		c.ato = minI64(c.ato, MAX_ATO)
		// initial cwnd
		c.swnd = calSwnd(c.rtt) >> 1
		c.cwnd = 8
		go c.internalRecvLoop()
		go c.internalSendLoop()
		go c.internalAckLoop()
		if debug >= 0 {
			go c.internal_state()
		}
		return nil
	} else {
		return ErrUnknown
	}
}

func (c *Conn) initDialing() error {
	// first syn
	pk := &packet{
		seq:  c.mySeq,
		flag: F_SYN,
	}
	item := nodeOf(pk)
	var buf []byte
	c.state = S_SYN0
	t0 := Now()
	for i := 0; i < MAX_RETRIES && c.state == S_SYN0; i++ {
		// send syn
		c.internalWrite(item)
		select {
		case buf = <-c.evRecv:
			c.rtt = Now() - t0
			c.state = S_SYN1
			c.connId.setRid(buf)
			buf = buf[TH_SIZE:]
		case <-time.After(time.Second):
			continue
		}
	}
	if c.state == S_SYN0 {
		return ErrTooManyAttempts
	}

	unmarshall(pk, buf)
	// expected syn+ack
	if pk.flag == F_SYN|F_ACK && pk.ack == c.mySeq {
		if scnt := pk.scnt - 1; scnt > 0 {
			c.rtt -= int64(scnt) * 1e3
		}
		log.Println("rtt", c.rtt)
		c.state = S_EST0
		// build ack3
		pk.scnt = 0
		pk.ack = pk.seq
		pk.flag = F_ACK
		item := nodeOf(pk)
		// send ack3
		c.internalWrite(item)
		// update lastAck
		c.logAck(pk.ack)
		c.state = S_EST1
		return nil
	} else {
		return ErrInexplicableData
	}
}

func (c *Conn) acceptConnection(buf []byte) error {
	var pk = new(packet)
	var item *qNode
	unmarshall(pk, buf)
	// expected syn
	if pk.flag == F_SYN {
		c.state = S_SYN1
		// build syn+ack
		pk.ack = pk.seq
		pk.seq = c.mySeq
		pk.flag |= F_ACK
		// update lastAck
		c.logAck(pk.ack)
		item = nodeOf(pk)
		item.scnt = pk.scnt - 1
	} else {
		dumpb("Syn1 ?", buf)
		return ErrInexplicableData
	}
	for i := 0; i < 5 && c.state == S_SYN1; i++ {
		t0 := Now()
		// reply syn+ack
		c.internalWrite(item)
		// recv ack3
		select {
		case buf = <-c.evRecv:
			c.state = S_EST0
			c.rtt = Now() - t0
			buf = buf[TH_SIZE:]
			log.Println("rtt", c.rtt)
		case <-time.After(time.Second):
			continue
		}
	}
	if c.state == S_SYN1 {
		return ErrTooManyAttempts
	}

	pk = new(packet)
	unmarshall(pk, buf)
	// expected ack3
	if pk.flag == F_ACK && pk.ack == c.mySeq {
		c.state = S_EST1
	} else {
		// if ack3 lost, resend syn+ack 3-times
		// and drop these coming data
		if pk.flag&F_DATA != 0 && pk.seq > c.lastAck {
			c.internalWrite(item)
			c.state = S_EST1
		} else {
			dumpb("Ack3 ?", buf)
			return ErrInexplicableData
		}
	}
	return nil
}

// 10,10,10, 100,100,100, 100,100,100, 1s,1s,1s
func selfSpinWait(fn func() bool) error {
	const MAX_SPIN = 12
	for i := 0; i < MAX_SPIN; i++ {
		if fn() {
			return nil
		} else if i <= 3 {
			time.Sleep(_10ms)
		} else if i <= 9 {
			time.Sleep(_100ms)
		} else {
			time.Sleep(time.Second)
		}
	}
	return ErrIOTimeout
}

/*
active close:
1 <- send fin-W: closeW()
	 before sending, ensure all outQ items has beed sent out and all of them has been acked.
2 -> wait to recv ack{fin-W}
	 then trigger closeR, including send fin-R and wait to recv ack{fin-R}

passive close:
-> fin:
	if outQ is not empty then self-spin wait.
	if outQ empty, send ack{fin-W} then goto closeW().
*/
func (c *Conn) Close() (err error) {
	if !atomic.CompareAndSwapInt32(&c.state, S_EST1, S_FIN0) {
		return selfSpinWait(func() bool {
			return atomic.LoadInt32(&c.state) == S_FIN
		})
	}
	var err0 error
	err0 = c.closeW()
	// waiting for fin-2 of peer
	err = selfSpinWait(func() bool {
		select {
		case v := <-c.evClose:
			if v == S_FIN {
				return true
			} else {
				time.AfterFunc(_100ms, func() { c.evClose <- v })
			}
		default:
		}
		return false
	})
	defer c.afterShutdown()
	if err != nil {
		// backup path for wait ack(finW) timeout
		c.closeR(nil)
	}
	// now could exit evRecv
	c.evRecv <- nil
	if err0 != nil {
		return err0
	} else {
		return
	}
}

func (c *Conn) beforeCloseW() (err error) {
	// check outQ was empty and all has been acked.
	// self-spin waiting
	for i := 0; i < 2; i++ {
		err = selfSpinWait(func() bool {
			return atomic.LoadInt32(&c.outPending) <= 0
		})
		if err == nil {
			break
		}
	}
	// send fin, reliably
	c.outlock.Lock()
	c.mySeq++
	c.outPending++
	pk := &packet{seq: c.mySeq, flag: F_FIN}
	item := nodeOf(pk)
	c.outQ.appendTail(item)
	c.internalWrite(item)
	c.outlock.Unlock()
	c.evSWnd <- VSWND_ACTIVE
	return
}

func (c *Conn) closeW() (err error) {
	defer c.afterCloseW()
	err = c.beforeCloseW()
	// waiting for outQ means:
	// 1. all outQ has been acked, for passive
	// 2. fin has been acked, for active
	var closed bool
	var max = 20
	if c.rtt > 200 {
		max = int(c.rtt) / 10
	}
	for i := 0; i < max && (atomic.LoadInt32(&c.outPending) > 0 || !closed); i++ {
		select {
		case v := <-c.evClose:
			if v == S_FIN0 {
				// namely, last fin has been acked.
				closed = true
			} else {
				time.AfterFunc(_100ms, func() { c.evClose <- v })
			}
		case <-time.After(_100ms):
		}
	}
	if closed || err != nil {
		return
	} else {
		return ErrIOTimeout
	}
}

func (c *Conn) afterCloseW() {
	// don't close evRecv to avoid endpoint dispatch exception
	//close(c.evRecv)
	// stop pending inputAndSend
	select {
	case c.evSend <- _CLOSE:
	default:
	}
	// stop internalSendLoop
	c.evSWnd <- _CLOSE
}

func (c *Conn) afterShutdown() {
	c.edp.removeConn(c.connId)
	log.Println("shutdown", c.state)
}

func (c *Conn) fakeShutdown() {
	select {
	case c.evClose <- S_FIN0:
	default:
	}
}

func (c *Conn) closeR(pk *packet) {
	var passive = true
	for {
		switch state := atomic.LoadInt32(&c.state); state {
		case S_FIN:
			return
		case S_FIN1: // multiple FIN, maybe lost
			c.passiveCloseReply(pk, false)
			return
		case S_FIN0: // active close preformed
			passive = false
			fallthrough
		default:
			if !atomic.CompareAndSwapInt32(&c.state, state, S_FIN1) {
				continue
			}
			c.passiveCloseReply(pk, true)
		}
		break
	}
	// here, R is closed.
	// ^^^^^^^^^^^^^^^^^^^^^
	if passive {
		// passive closing call closeW contains sending fin and recv ack
		// may the ack of fin-2 was lost, then the closeW will timeout
		c.closeW()
	}
	// here, R,W both were closed.
	c.state = S_FIN
	// no need for recv then stop internalAckLoop
	c.evAck <- _CLOSE
	if passive {
		c.afterShutdown()
	} else {
		// notify active close thread
		select {
		case c.evClose <- S_FIN:
		default:
		}
	}
}

func (c *Conn) passiveCloseReply(pk *packet, first bool) {
	if pk != nil && pk.flag&F_FIN != 0 {
		if first {
			c.checkInQ(pk)
			close(c.evRead)
		}
		// ack the FIN
		pk = &packet{seq: _FIN_ACK_SEQ, ack: pk.seq, flag: F_ACK}
		item := nodeOf(pk)
		c.internalWrite(item)
	}
}

// check inQ ends orderly, and copy queue data to user space
func (c *Conn) checkInQ(pk *packet) {
	if nil != selfSpinWait(func() bool {
		return c.inMaxCtnSeq+1 == pk.seq
	}) { // timeout for waiting inQ to finish
		return
	}
	c.inlock.Lock()
	defer c.inlock.Unlock()
	if c.inQ.size() > 0 {
		for i := c.inQ.head; i != nil; i = i.next {
			c.inQReady = append(c.inQReady, i.payload...)
		}
	}
}
