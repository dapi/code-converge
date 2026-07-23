//go:build !windows

package runner

import (
	"bytes"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// copyStreams is the one observer-side reader for both process streams.
// Polling readiness in one goroutine avoids making observer order a function
// of two reader goroutines being scheduled. It cannot establish the order of
// writes made to independent stdout/stderr pipes.
func copyStreams(stdoutReader, stderrReader *os.File, stdout, stderr *bytes.Buffer, outputs *outputQueue, processDone <-chan struct{}) {
	streams := []struct {
		file    *os.File
		source  string
		capture *bytes.Buffer
	}{
		{file: stdoutReader, source: "stdout", capture: stdout},
		{file: stderrReader, source: "stderr", capture: stderr},
	}
	closed := [2]bool{}
	processExited := false
	buffer := make([]byte, 32*1024)

	for {
		if !processExited {
			select {
			case <-processDone:
				processExited = true
			default:
			}
		}
		if closed[0] && closed[1] {
			return
		}

		fds := make([]unix.PollFd, 0, 2)
		indices := make([]int, 0, 2)
		for i, stream := range streams {
			if !closed[i] {
				fds = append(fds, unix.PollFd{Fd: int32(stream.file.Fd()), Events: unix.POLLIN})
				indices = append(indices, i)
			}
		}
		// Keep the poll interruptible so processDone can switch the mux to its
		// post-exit behavior even when the child produced no output. A bounded
		// observer drain can race publication of final buffered bytes, so retain
		// the short timeout while it is active.
		timeout := 50
		if _, err := unix.Poll(fds, timeout); err != nil {
			if err == unix.EINTR {
				continue
			}
			return
		}
		ready := false
		for fdIndex, pollfd := range fds {
			if pollfd.Revents&(unix.POLLIN|unix.POLLHUP|unix.POLLERR) == 0 {
				continue
			}
			ready = true
			streamIndex := indices[fdIndex]
			stream := streams[streamIndex]
			n, err := stream.file.Read(buffer)
			if n > 0 {
				chunk := append([]byte(nil), buffer[:n]...)
				_, _ = stream.capture.Write(chunk)
				if outputs != nil {
					outputs.add(Output{Source: stream.source, Data: chunk})
				}
			}
			if err != nil {
				if err == io.EOF {
					closed[streamIndex] = true
				}
			}
		}
		if processExited && outputs != nil && !ready {
			return
		}
	}
}
