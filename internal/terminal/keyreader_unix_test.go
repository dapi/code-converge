//go:build !windows

package terminal

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestPollingKeyReaderDoesNotChangeOriginalTerminalFlags(t *testing.T) {
	original, err := os.CreateTemp(t.TempDir(), "input")
	if err != nil {
		t.Fatal(err)
	}
	defer original.Close()
	sharedFD, err := unix.Dup(int(original.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	sharedOutput := os.NewFile(uintptr(sharedFD), "shared-output")
	defer sharedOutput.Close()

	previousOpen := openKeyReaderInput
	openKeyReaderInput = func() (*os.File, error) {
		return os.OpenFile(original.Name(), os.O_RDONLY, 0)
	}
	t.Cleanup(func() { openKeyReaderInput = previousOpen })

	before, err := unix.FcntlInt(original.Fd(), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := newKeyReader(original)
	if err != nil {
		t.Fatal(err)
	}
	after, err := unix.FcntlInt(sharedOutput.Fd(), unix.F_GETFL, 0)
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("shared stdout flags changed: got %#x, want %#x", after, before)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
}
