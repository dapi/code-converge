//go:build windows

package runner

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"
)

const windowsStreamDrainTimeout = 50 * time.Millisecond

// Windows does not provide a portable poll operation for anonymous pipes.
// Keep capture functional there; the Unix implementation provides the
// Windows does not provide a portable poll operation for anonymous pipes.
// Keep capture functional there; source labels are preserved but exact
// cross-stream write order is not available from independent pipes.
func copyStreams(stdoutReader, stderrReader *os.File, stdout, stderr *bytes.Buffer, outputs *outputQueue, processDone <-chan struct{}) {
	var wg sync.WaitGroup
	if outputs != nil {
		// Windows anonymous-pipe reads block until a writer closes its handle.
		// A descendant can inherit that handle after the direct child exits, so
		// bound only the explicitly lossy live-observer path.
		go func() {
			<-processDone
			deadline := time.Now().Add(windowsStreamDrainTimeout)
			if err := stdoutReader.SetReadDeadline(deadline); err != nil {
				_ = stdoutReader.Close()
			}
			if err := stderrReader.SetReadDeadline(deadline); err != nil {
				_ = stderrReader.Close()
			}
		}()
	}
	copy := func(source string, reader io.Reader, capture *bytes.Buffer) {
		defer wg.Done()
		buffer := make([]byte, 32*1024)
		for {
			n, err := reader.Read(buffer)
			if n > 0 {
				chunk := append([]byte(nil), buffer[:n]...)
				_, _ = capture.Write(chunk)
				if outputs != nil {
					outputs.add(Output{Source: source, Data: chunk})
				}
			}
			if err != nil {
				return
			}
		}
	}
	wg.Add(2)
	go copy("stdout", stdoutReader, stdout)
	go copy("stderr", stderrReader, stderr)
	wg.Wait()
}
