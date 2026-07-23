package runner

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Invocation struct {
	Executable string
	Args       []string
	Stdin      string
	Env        []string
	Output     func(Output)
}

// Output is one observed process chunk. It is delivered while a process is
// running and remains separate from the final captured Result. Supplying an
// observer opts into the bounded live-observer drain; leave it nil when the
// caller requires inherited-descriptor output to be captured through EOF.
// Chunks retain their source label, but independent stdout/stderr pipes do not
// provide an exact cross-stream write order.
type Output struct {
	Source string // stdout or stderr
	Data   []byte
}

// StageContext identifies a Codex-backed workflow stage for private
// diagnostics. It is deliberately separate from Invocation so the process
// boundary stays usable for non-workflow commands such as git and gh.
type StageContext struct {
	Stage           string
	ReviewPhase     int
	Cycle           int
	Model           string
	ReasoningEffort string
}

type stageContextKey struct{}

func WithStageContext(ctx context.Context, stage StageContext) context.Context {
	return context.WithValue(ctx, stageContextKey{}, stage)
}

func StageContextFrom(ctx context.Context) (StageContext, bool) {
	stage, ok := ctx.Value(stageContextKey{}).(StageContext)
	return stage, ok && stage.Stage != ""
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Runner interface {
	Run(context.Context, Invocation) (Result, error)
}

type Exec struct {
	Executable string
	Dir        string
}

type outputQueue struct {
	mu           sync.Mutex
	cond         *sync.Cond
	items        []Output
	pendingBytes int
	closed       bool
}

// The observer is a presentation path and must not be allowed to retain an
// unbounded amount of process output when rendering falls behind. Captured
// stdout/stderr remain lossless; only the pending observer data is lossy under
// sustained overload. Evicting the oldest data keeps the view close to the
// live tail, which is also what the bounded panes display.
const (
	maxOutputQueueBytes = 1 << 20
	maxOutputQueueItems = 32
)

func newOutputQueue() *outputQueue {
	q := &outputQueue{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

func (q *outputQueue) add(output Output) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}

	if len(output.Data) > maxOutputQueueBytes {
		output.Data = output.Data[len(output.Data)-maxOutputQueueBytes:]
	}
	for len(q.items) > 0 && (len(q.items) >= maxOutputQueueItems || q.pendingBytes+len(output.Data) > maxOutputQueueBytes) {
		q.pendingBytes -= len(q.items[0].Data)
		q.items[0].Data = nil
		q.items = q.items[1:]
	}
	q.items = append(q.items, output)
	q.pendingBytes += len(output.Data)
	q.cond.Signal()
}

func (q *outputQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.cond.Broadcast()
	q.mu.Unlock()
}

func (q *outputQueue) run(output func(Output)) {
	for {
		q.mu.Lock()
		for len(q.items) == 0 && !q.closed {
			q.cond.Wait()
		}
		if len(q.items) == 0 && q.closed {
			q.mu.Unlock()
			return
		}
		item := q.items[0]
		q.pendingBytes -= len(item.Data)
		q.items[0].Data = nil
		q.items = q.items[1:]
		q.mu.Unlock()
		output(item)
	}
}

func (r Exec) Run(ctx context.Context, invocation Invocation) (Result, error) {
	name := r.Executable
	if invocation.Executable != "" {
		name = invocation.Executable
	}
	if name == "" {
		name = "codex"
	}
	cmd := exec.Command(name, invocation.Args...)
	configureProcessGroup(cmd)
	cmd.Dir = r.Dir
	cmd.Env = append(os.Environ(), invocation.Env...)
	cmd.Stdin = bytes.NewBufferString(invocation.Stdin)
	var stdout, stderr bytes.Buffer
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("capture stdout: %w", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		return Result{}, fmt.Errorf("capture stderr: %w", err)
	}
	cmd.Stdout = stdoutWriter
	cmd.Stderr = stderrWriter
	if err := ctx.Err(); err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		_ = stderrReader.Close()
		_ = stderrWriter.Close()
		return Result{ExitCode: -1}, err
	}
	err = cmd.Start()
	// The child (and any descendants it creates) now owns the write ends.
	// Keeping parent copies open would prevent the readers from seeing EOF.
	_ = stdoutWriter.Close()
	_ = stderrWriter.Close()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stderrReader.Close()
		result := Result{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: -1}
		return result, formatRunError(name, err, result.Stderr)
	}
	var outputDone chan struct{}
	var outputs *outputQueue
	if invocation.Output != nil {
		outputs = newOutputQueue()
		outputDone = make(chan struct{})
		go func() {
			outputs.run(invocation.Output)
			close(outputDone)
		}()
	}
	done := make(chan struct{})
	streamsDone := make(chan struct{})
	// A single multiplexer observes both descriptors. Separate reader
	// goroutines would make observer order depend on goroutine scheduling.
	go func() {
		copyStreams(stdoutReader, stderrReader, &stdout, &stderr, outputs, done)
		close(streamsDone)
	}()
	go func() {
		select {
		case <-ctx.Done():
			terminateProcessGroup(cmd)
		case <-done:
		}
	}()
	err = cmd.Wait()
	close(done)
	// Capture-only runs drain to EOF so descendants that inherited the pipe do
	// not lose output. The live observer path uses a bounded post-exit drain to
	// avoid making an interactive run wait forever for a descendant-owned pipe.
	<-streamsDone
	_ = stdoutReader.Close()
	_ = stderrReader.Close()
	if outputs != nil {
		outputs.close()
		<-outputDone
	}
	result := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result, formatRunError(name, err, result.Stderr)
	}
	return result, nil
}

func formatRunError(name string, err error, stderr string) error {
	detail := strings.TrimSpace(stderr)
	if len(detail) > 8192 {
		detail = detail[:8192] + "…"
	}
	if detail != "" {
		return fmt.Errorf("%s exited unsuccessfully: %w: %s", name, err, detail)
	}
	return fmt.Errorf("%s exited unsuccessfully: %w", name, err)
}
