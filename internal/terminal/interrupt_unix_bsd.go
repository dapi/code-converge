//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package terminal

import "golang.org/x/sys/unix"

func getTerminalTermiosOS(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TIOCGETA)
}

func setTerminalTermiosOS(fd int, termios *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TIOCSETA, termios)
}
