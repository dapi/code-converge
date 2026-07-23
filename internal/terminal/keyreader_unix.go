//go:build !windows

package terminal

import (
	"sync/atomic"

	"golang.org/x/sys/unix"
)

type keyReader interface {
	Read([]byte) (int, error)
	Close() error
}

type pollingKeyReader struct {
	in       interface{ Read([]byte) (int, error) }
	fd       int
	original int
	closed   atomic.Bool
}

func newKeyReader(in interface {
	Fd() uintptr
	Read([]byte) (int, error)
}) (keyReader, error) {
	fd := int(in.Fd())
	original, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		return nil, err
	}
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFL, original|unix.O_NONBLOCK); err != nil {
		return nil, err
	}
	return &pollingKeyReader{in: in, fd: fd, original: original}, nil
}

func (r *pollingKeyReader) Read(buf []byte) (int, error) {
	for {
		if r.closed.Load() {
			return 0, unix.EBADF
		}
		ready, err := unix.Poll([]unix.PollFd{{Fd: int32(r.fd), Events: unix.POLLIN}}, 50)
		if err != nil {
			return 0, err
		}
		if ready == 0 {
			continue
		}
		return r.in.Read(buf)
	}
}

func (r *pollingKeyReader) Close() error {
	if r.closed.Swap(true) {
		return nil
	}
	_, err := unix.FcntlInt(uintptr(r.fd), unix.F_SETFL, r.original)
	return err
}
