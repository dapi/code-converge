//go:build windows

package terminal

import (
	"os"
	"sync"
	"time"
)

type resizeSignal struct{}

func (resizeSignal) Signal()        {}
func (resizeSignal) String() string { return "console resize" }

func resizeSignals() (<-chan os.Signal, func()) {
	ch := make(chan os.Signal, 1)
	stop := make(chan struct{})
	var stopped sync.WaitGroup
	stopped.Add(1)
	go func() {
		defer stopped.Done()
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				select {
				case ch <- resizeSignal{}:
				default:
				}
			}
		}
	}()
	return ch, func() {
		close(stop)
		stopped.Wait()
	}
}
