//go:build !windows

package terminal

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func TestEnableTerminalInterruptSetsISIG(t *testing.T) {
	previousGet, previousSet := getTerminalTermios, setTerminalTermios
	t.Cleanup(func() {
		getTerminalTermios, setTerminalTermios = previousGet, previousSet
	})
	termios := &unix.Termios{Lflag: unix.ECHO}
	getTerminalTermios = func(fd int) (*unix.Termios, error) {
		if fd != 17 {
			t.Fatalf("fd = %d, want 17", fd)
		}
		return termios, nil
	}
	setTerminalTermios = func(fd int, got *unix.Termios) error {
		if fd != 17 {
			t.Fatalf("fd = %d, want 17", fd)
		}
		if got.Lflag&unix.ISIG == 0 {
			t.Fatal("ISIG was not enabled")
		}
		return nil
	}

	if err := enableTerminalInterrupt(17); err != nil {
		t.Fatal(err)
	}
}

func TestEnableTerminalInterruptReturnsTerminalReadError(t *testing.T) {
	previousGet, previousSet := getTerminalTermios, setTerminalTermios
	t.Cleanup(func() {
		getTerminalTermios, setTerminalTermios = previousGet, previousSet
	})
	want := errors.New("read termios")
	getTerminalTermios = func(int) (*unix.Termios, error) { return nil, want }
	setTerminalTermios = func(int, *unix.Termios) error {
		t.Fatal("set terminal termios called after read failure")
		return nil
	}

	if err := enableTerminalInterrupt(17); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}
