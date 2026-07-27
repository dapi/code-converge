//go:build aix || linux || solaris || zos

package terminal

import "golang.org/x/sys/unix"

func getTerminalTermiosOS(fd int) (*unix.Termios, error) {
	return unix.IoctlGetTermios(fd, unix.TCGETS)
}

func setTerminalTermiosOS(fd int, termios *unix.Termios) error {
	return unix.IoctlSetTermios(fd, unix.TCSETS, termios)
}
