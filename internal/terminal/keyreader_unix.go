//go:build !windows

package terminal

import (
	"os"
	"sync/atomic"

	"golang.org/x/sys/unix"
)

type keyReader interface {
	Read([]byte) (int, error)
	Close() error
}

type pollingKeyReader struct {
	in       *os.File
	fd       int
	original int
	closed   atomic.Bool
}

var openKeyReaderInput = func() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDONLY, 0)
}

func newKeyReader(_ interface {
	Fd() uintptr
	Read([]byte) (int, error)
}) (keyReader, error) {
	// O_NONBLOCK is an open-file-description flag. A dup of stdin would still
	// share it with stdout when their descriptors refer to the same terminal,
	// so reopen the controlling terminal for the polling reader instead.
	in, err := openKeyReaderInput()
	if err != nil {
		return nil, err
	}
	fd := int(in.Fd())
	original, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		_ = in.Close()
		return nil, err
	}
	if _, err := unix.FcntlInt(uintptr(fd), unix.F_SETFL, original|unix.O_NONBLOCK); err != nil {
		_ = in.Close()
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
	closeErr := r.in.Close()
	if err != nil {
		return err
	}
	return closeErr
}
