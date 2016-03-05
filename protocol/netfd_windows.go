package suft

import (
	"net"
	"syscall"
)

// Network file descriptor.
type netFD struct {
	// locking/lifetime of sysfd + serialize access to Read and Write methods
	fdmu fdMutex

	// immutable until Close
	sysfd         syscall.Handle
	family        int
	sotype        int
	isConnected   bool
	skipSyncNotif bool
	net           string
	laddr         net.Addr
	raddr         net.Addr

	rop operation // read operation
	wop operation // write operation

	// wait server
	pd pollDesc
}

// operation contains superset of data necessary to perform all async IO.
type operation struct {
	// Used by IOCP interface, it must be first field
	// of the struct, as our code rely on it.
	o syscall.Overlapped

	// fields used by runtime.netpoll
	runtimeCtx uintptr
	mode       int32
	errno      int32
	qty        uint32

	// fields used only by net package
	fd     *netFD
	errc   chan error
	buf    syscall.WSABuf
	sa     syscall.Sockaddr
	rsa    *syscall.RawSockaddrAny
	rsan   int32
	handle syscall.Handle
	flags  uint32
}
