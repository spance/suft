// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package suft

import (
	"net"
)

// Network file descriptor.
type netFD struct {
	// locking/lifetime of sysfd + serialize access to Read and Write methods
	fdmu fdMutex

	// immutable until Close
	sysfd       int
	family      int
	sotype      int
	isConnected bool
	net         string
	laddr       net.Addr
	raddr       net.Addr

	// wait server
	pd pollDesc
}
