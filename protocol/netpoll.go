package suft

import (
	"net"
	"unsafe"
)

func net_pollSetDeadline(ctx uintptr, d int64, mode int)

type fdMutex struct {
	state uint64
	rsema uint32
	wsema uint32
}

type pollDesc struct {
	runtimeCtx uintptr
}

type netConn struct {
	fd *netFD
}

func getPollCtx(c *net.UDPConn) uintptr {
	connPtr := (*netConn)(unsafe.Pointer(c))
	return connPtr.fd.pd.runtimeCtx
}
