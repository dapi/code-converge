package terminal

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/term"
)

func TestRawModeReportsTerminalState(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	if v.RawMode() {
		t.Fatal("new view unexpectedly reports raw mode")
	}
	v.restore = &term.State{}
	if !v.RawMode() {
		t.Fatal("view with saved terminal state does not report raw mode")
	}
}

func TestSanitize(t *testing.T) {
	got := Sanitize([]byte("one\x1b[31mtwo\x1b[0m\r\nthree\x00\x9b"))
	if got != "onetwo\nthree�" {
		t.Fatalf("Sanitize = %q", got)
	}
}

func TestViewKeyNavigation(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.open = true
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

func TestViewIgnoresNavigationWhileClosed(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.workflowOffset = 4
	v.agentOffset = 5
	v.handleKeys([]byte{'\t', 0x1b, '[', 'A', 0x1b, '[', '5', '~', 0x1b, '[', 'F'})

	v.mu.Lock()
	focus, workflowOffset, agentOffset := v.focus, v.workflowOffset, v.agentOffset
	v.mu.Unlock()
	if focus != 0 || workflowOffset != 4 || agentOffset != 5 {
		t.Fatalf("closed navigation changed focus=%d workflowOffset=%d agentOffset=%d", focus, workflowOffset, agentOffset)
	}
}

func TestViewReplacesAgentStream(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.StartAgent("review 1")
	v.AppendAgent("stdout", []byte("first"))
	v.StartAgent("fix 1")
	v.AppendAgent("stdout", []byte("second"))
	v.CompleteAgent("completed")
	v.mu.Lock()
	got := strings.Join(v.agent, "\n")
	v.mu.Unlock()
	if strings.Contains(got, "first") || !strings.Contains(got, "[fix 1]") || !strings.Contains(got, "second") {
		t.Fatalf("agent stream = %q", got)
	}
}

func TestNewAgentStreamReturnsToLiveTail(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.agentOffset = 99
	if err := v.StartAgent("review 2"); err != nil {
		t.Fatal(err)
	}
	if v.agentOffset != 0 {
		t.Fatalf("agent offset = %d, want live tail", v.agentOffset)
	}
}

func TestViewReassemblesChunkedStreamsIntoLogicalLines(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.StartAgent("review 1")
	v.AppendAgent("stdout", []byte("hel"))
	v.AppendAgent("stdout", []byte("lo\nnext"))
	v.AppendAgent("stderr", []byte("one\ntwo\n"))
	v.AppendAgent("stdout", []byte(" "))
	v.AppendAgent("stdout", []byte("\xe2\x82"))
	v.AppendAgent("stdout", []byte("\xac\n"))
	v.CompleteAgent("completed")
	v.mu.Lock()
	got := strings.Join(v.agent, "\n")
	v.mu.Unlock()
	for _, want := range []string{"hello", "next €", "[stderr] one", "[stderr] two"} {
		if !strings.Contains(got, want) {
			t.Fatalf("agent stream = %q, missing %q", got, want)
		}
	}
}

func TestPendingStreamMetadataIsBounded(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	for i := 0; i < 10_000; i++ {
		if err := v.AppendAgent("stdout", []byte("line\n")); err != nil {
			t.Fatal(err)
		}
		if err := v.AppendAgent("stderr", []byte("line\n")); err != nil {
			t.Fatal(err)
		}
	}
	if len(v.pendingOrder) > 2 {
		t.Fatalf("pending order grew to %d entries", len(v.pendingOrder))
	}
}

func TestUnterminatedStreamIsBounded(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.StartAgent("review 1")
	data := bytes.Repeat([]byte{'x'}, maxPendingBytes*4)
	if err := v.AppendAgent("stdout", data); err != nil {
		t.Fatal(err)
	}
	if got := len(v.agentPending["stdout"]); got > maxPendingBytes {
		t.Fatalf("pending bytes = %d, want <= %d", got, maxPendingBytes)
	}
}

func TestCtrlCForwardsInterrupt(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	called := 0
	v.Interrupt = func() { called++ }
	v.handleKeys([]byte{0x03})
	if called != 1 {
		t.Fatalf("interrupt calls = %d", called)
	}
}

func TestStopClosesKeyReaderBeforeWaiting(t *testing.T) {
	reader := &blockingKeyReader{released: make(chan struct{})}
	v := New(&bytes.Buffer{}, nil)
	v.reader = reader
	v.stop = make(chan struct{})
	v.done = make(chan struct{})
	go v.readKeys(v.stop, reader, v.done)

	done := make(chan struct{})
	go func() {
		_ = v.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Stop waited for a blocked key reader")
	}
	if !reader.closed() {
		t.Fatal("key reader was not closed")
	}
}

type blockingKeyReader struct {
	once     sync.Once
	released chan struct{}
}

func (r *blockingKeyReader) Read([]byte) (int, error) {
	<-r.released
	return 0, errors.New("closed")
}

func (r *blockingKeyReader) Close() error {
	r.once.Do(func() { close(r.released) })
	return nil
}

func (r *blockingKeyReader) closed() bool {
	select {
	case <-r.released:
		return true
	default:
		return false
	}
}

func TestRenderFitsTerminalAndPropagatesWriteFailure(t *testing.T) {
	var out bytes.Buffer
	v := New(&out, nil)
	v.open = true
	if err := v.Resize(); err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(out.String(), "\n") + 1; lines != 24 {
		t.Fatalf("rendered %d rows, want 24: %q", lines, out.String())
	}
	failing := New(errorWriter{}, nil)
	failing.open = true
	if err := failing.Resize(); !errors.Is(err, errWrite) {
		t.Fatalf("render error = %v", err)
	}
	if err := failing.Err(); !errors.Is(err, errWrite) {
		t.Fatalf("stored render error = %v", err)
	}
}

var errWrite = errors.New("write failed")

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errWrite }

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

func TestViewPreservesCrossStreamArrivalOrderForPartialLines(t *testing.T) {
	v := New(&bytes.Buffer{}, nil)
	v.StartAgent("review 1")
	v.AppendAgent("stdout", []byte("out-"))
	v.AppendAgent("stderr", []byte("err\n"))
	v.AppendAgent("stdout", []byte("done\n"))

	v.mu.Lock()
	got := strings.Join(v.agent, "\n")
	v.mu.Unlock()
	outIndex := strings.Index(got, "out-done")
	errIndex := strings.Index(got, "[stderr] err")
	if outIndex < 0 || errIndex < 0 || outIndex > errIndex {
		t.Fatalf("agent stream order = %q", got)
	}
}
