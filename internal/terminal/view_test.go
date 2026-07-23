package terminal

import (
	"bytes"
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	got := Sanitize([]byte("one\x1b[31mtwo\x1b[0m\r\nthree\x00\x9b"))
	if got != "onetwo\nthree�" {
		t.Fatalf("Sanitize = %q", got)
	}
}

func TestViewKeyNavigation(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.workflow = []string{"one", "two", "three"}
	v.handleKeys([]byte{'\t', 0x1b, '['})
	v.handleKeys([]byte{'A', 0x1b, '[', 'F'})
	v.mu.Lock()
	focus, offset := v.focus, v.agentOffset
	v.mu.Unlock()
	if focus != 1 || offset != 0 {
		t.Fatalf("focus=%d agentOffset=%d", focus, offset)
	}
}

func TestViewReplacesAgentStream(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.StartAgent("review 1")
	v.AppendAgent("stdout", []byte("first"))
	v.StartAgent("fix 1")
	v.AppendAgent("stdout", []byte("second"))
	v.mu.Lock()
	got := strings.Join(v.agent, "\n")
	v.mu.Unlock()
	if strings.Contains(got, "first") || !strings.Contains(got, "[fix 1]") || !strings.Contains(got, "second") {
		t.Fatalf("agent stream = %q", got)
	}
}

func TestViewBoundsAndSeparatesStreams(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.StartAgent("review 1")
	v.AppendAgent("stdout", []byte("out\n"))
	v.AppendAgent("stderr", []byte("err\n"))
	v.mu.Lock()
	got := strings.Join(v.agent, "\n")
	v.mu.Unlock()
	if !strings.Contains(got, "out") || !strings.Contains(got, "[stderr] err") {
		t.Fatalf("agent = %q", got)
	}
	for i := 0; i < maxLines+10; i++ {
		v.AppendWorkflow("line")
	}
	v.mu.Lock()
	n := len(v.workflow)
	v.mu.Unlock()
	if n != maxLines {
		t.Fatalf("workflow lines = %d", n)
	}
}
