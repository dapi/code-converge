//go:build !windows

package terminal

import "golang.org/x/sys/unix"

var (
	getTerminalTermios = getTerminalTermiosOS
	setTerminalTermios = setTerminalTermiosOS
)

// enableTerminalInterrupt preserves Ctrl-C's normal SIGINT behavior after
// MakeRaw disables ISIG for single-key input handling.
func enableTerminalInterrupt(fd int) error {
	termios, err := getTerminalTermios(fd)
	if err != nil {
		return err
	}
	termios.Lflag |= unix.ISIG
	return setTerminalTermios(fd, termios)
}
