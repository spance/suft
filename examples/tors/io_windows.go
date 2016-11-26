package main

import (
	"io"
	"log"
	"os"
	"syscall"
	"unsafe"
)

var (
	_CreateEvent         *syscall.Proc
	_GetOverlappedResult *syscall.Proc
)

func init() {
	wk, _ := syscall.LoadDLL("kernel32.dll")
	_CreateEvent = wk.MustFindProc("CreateEventW")
	_GetOverlappedResult = wk.MustFindProc("GetOverlappedResult")
}

func getStdin() io.Reader {
	return newWinAsyncIOFile(os.Stdin.Fd())
}

func getStdout() io.Writer {
	return newWinAsyncIOFile(os.Stdout.Fd())
}

type winAsyncIOFile struct {
	hd         syscall.Handle
	he         syscall.Handle
	overlapped syscall.Overlapped
}

func newWinAsyncIOFile(h uintptr) *winAsyncIOFile {
	s := &winAsyncIOFile{hd: syscall.Handle(h)}
	s.he = createEvent()
	s.overlapped.HEvent = s.he
	return s
}

func createEvent() syscall.Handle {
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms682396(v=vs.85).aspx
	h, _, e := _CreateEvent.Call(0, 0, 0, 0)
	if h == 0 { // return NULL
		log.Fatalln("_CreateEventW", e)
	}
	return syscall.Handle(h)
}

func (s *winAsyncIOFile) getOverlappedResult(done *uint32) error {
	// https://msdn.microsoft.com/en-us/library/windows/desktop/ms683209(v=vs.85).aspx
	r, _, en := syscall.Syscall6(_GetOverlappedResult.Addr(), 4,
		uintptr(s.hd),
		uintptr(unsafe.Pointer(&s.overlapped)),
		uintptr(unsafe.Pointer(done)),
		1,
		0, 0)
	if r != 0 {
		return nil
	} else {
		// 995=The I/O operation has been aborted because of either a thread exit or an application request.
		if en == syscall.ERROR_OPERATION_ABORTED {
			return io.EOF
		} else {
			return en
		}
	}
}

func (s *winAsyncIOFile) Read(b []byte) (int, error) {
	var done uint32
	e := syscall.ReadFile(s.hd, b, &done, &s.overlapped)
	if en, y := e.(syscall.Errno); y {
		if en == syscall.ERROR_IO_PENDING {
			e = s.getOverlappedResult(&done)
		}
	}
	return int(done), e
}

func (s *winAsyncIOFile) Write(b []byte) (int, error) {
	var done uint32
	e := syscall.WriteFile(s.hd, b, &done, &s.overlapped)
	if en, y := e.(syscall.Errno); y {
		if en == syscall.ERROR_IO_PENDING {
			e = s.getOverlappedResult(&done)
		}
	}
	return int(done), e
}

func (s *winAsyncIOFile) Close() error {
	return syscall.CancelIoEx(s.hd, &s.overlapped)
}
