//go:build windows

package terminal

import (
	"errors"
	"sync/atomic"

	"golang.org/x/sys/windows"
)

var errKeyReaderClosed = errors.New("key reader closed")

type keyReader interface {
	Read([]byte) (int, error)
	Close() error
}

type directKeyReader struct {
	in     interface{ Read([]byte) (int, error) }
	handle windows.Handle
	closed atomic.Bool
}

func newKeyReader(in interface {
	Fd() uintptr
	Read([]byte) (int, error)
}) (keyReader, error) {
	return &directKeyReader{in: in, handle: windows.Handle(in.Fd())}, nil
}

func (r *directKeyReader) Read(buf []byte) (int, error) {
	if r.closed.Load() {
		return 0, errKeyReaderClosed
	}
	for {
		if r.closed.Load() {
			return 0, errKeyReaderClosed
		}
		status, err := windows.WaitForSingleObject(r.handle, 50)
		if err != nil {
			return 0, err
		}
		if status == uint32(windows.WAIT_TIMEOUT) {
			continue
		}
		if status != windows.WAIT_OBJECT_0 {
			return 0, errors.New("console input wait failed")
		}
		return r.in.Read(buf)
	}
}

func (r *directKeyReader) Close() error {
	r.closed.Store(true)
	return nil
}
