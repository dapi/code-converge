//go:build windows

package terminal

import "golang.org/x/sys/windows"

func enableTerminalInterrupt(fd int) error {
	handle := windows.Handle(fd)
	var mode uint32
	if err := windows.GetConsoleMode(handle, &mode); err != nil {
		return err
	}
	return windows.SetConsoleMode(handle, mode|windows.ENABLE_PROCESSED_INPUT)
}
