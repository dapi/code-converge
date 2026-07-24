// Package terminal owns the narrow interactive terminal surface used by the
// human progress renderer. It deliberately has no workflow knowledge.
package terminal

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/term"
)

const maxLines = 2000
const maxPendingBytes = 64 * 1024

// View stores the two independent pane histories and, while open, renders
// them in the terminal's alternate screen.
type View struct {
	Out io.Writer
	In  interface {
		Fd() uintptr
		Read([]byte) (int, error)
	}

	mu                sync.Mutex
	open              bool
	workflow          []string
	agent             []string
	agentRecords      []*agentLine
	agentPendingLines map[string]*agentLine
	stream            string
	focus             int
	workflowOffset    int
	agentOffset       int
	pending           []byte
	agentPending      map[string][]byte
	pendingOrder      []string
	restore           *term.State
	stop              chan struct{}
	done              chan struct{}
	reader            keyReader
	err               error
	transientClearer  func() error
	// Interrupt is invoked when Ctrl-C is received while raw mode is active.
	// MakeRaw disables ISIG, so forwarding this byte is necessary to preserve
	// the command's normal signal-driven cancellation behavior.
	Interrupt func()
}

type agentLine struct {
	source   string
	data     []byte
	complete bool
}

func New(out io.Writer, in interface {
	Fd() uintptr
	Read([]byte) (int, error)
}) *View {
	return &View{Out: out, In: in}
}

// Eligible is intentionally conservative: callers must additionally apply
// their product format and TERM policy.
func (v *View) Eligible() bool {
	return v != nil && v.In != nil && term.IsTerminal(int(v.In.Fd())) && IsTerminalWriter(v.Out)
}

// IsTerminalWriter verifies that a writer is backed by a real terminal, not
// merely a character device such as /dev/null.
func IsTerminalWriter(out io.Writer) bool {
	file, ok := out.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(int(file.Fd()))
}

// Start puts stdin into raw mode so one-key input is available. It is safe to
// call only after Eligible and returns an error without changing output if raw
// mode cannot be established.
func (v *View) Start() error {
	if !v.Eligible() {
		return nil
	}
	state, err := term.MakeRaw(int(v.In.Fd()))
	if err != nil {
		return err
	}
	v.mu.Lock()
	v.restore = state
	v.stop = make(chan struct{})
	v.done = make(chan struct{})
	reader, err := newKeyReader(v.In)
	if err != nil {
		v.restore = nil
		v.stop = nil
		v.done = nil
		v.mu.Unlock()
		_ = term.Restore(int(v.In.Fd()), state)
		return err
	}
	v.reader = reader
	done := v.done
	v.agentPending = make(map[string][]byte)
	v.agentPendingLines = make(map[string]*agentLine)
	v.pendingOrder = nil
	v.mu.Unlock()
	go v.readKeys(v.stop, reader, done)
	go v.watchResize(v.stop)
	return nil
}

// Stop closes the split view and restores stdin after the cancellable reader
// has exited.
func (v *View) Stop() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	stop, done, state, reader := v.stop, v.done, v.restore, v.reader
	v.stop, v.done, v.restore, v.reader = nil, nil, nil, nil
	wasOpen := v.open
	v.open = false
	v.mu.Unlock()
	if stop != nil {
		close(stop)
	}
	var result error
	// Read may be blocked in the platform reader. Close it before waiting so
	// the reader observes the shutdown and can signal done.
	if reader != nil {
		if err := reader.Close(); err != nil {
			// Preserve the cleanup error, but still wait for the reader below.
			result = err
		}
	}
	if done != nil {
		<-done
	}
	if wasOpen {
		if _, err := io.WriteString(v.Out, "\x1b[?25h\x1b[?1049l"); err != nil {
			result = err
		}
	}
	if state != nil {
		if err := term.Restore(int(v.In.Fd()), state); err != nil && result == nil {
			result = err
		}
	}
	return result
}

func (v *View) Active() bool {
	if v == nil {
		return false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.open
}

// RawMode reports whether Start has put the input terminal into raw mode.
// Raw mode disables the terminal's usual LF-to-CRLF expansion, so permanent
// records must explicitly return to column zero before their next line.
func (v *View) RawMode() bool {
	if v == nil {
		return false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.restore != nil
}

func (v *View) AppendWorkflow(line string) error {
	if v == nil || line == "" {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return v.err
	}
	v.workflow = appendBounded(v.workflow, splitLines(line)...)
	return v.renderLocked()
}

func (v *View) StartAgent(identity string) error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return v.err
	}
	v.stream = identity
	v.agent = []string{"[" + identity + "]"}
	v.agentRecords = nil
	v.agentOffset = 0
	v.agentPending = make(map[string][]byte)
	v.agentPendingLines = make(map[string]*agentLine)
	v.pendingOrder = nil
	return v.renderLocked()
}

func (v *View) CompleteAgent(state string) error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return v.err
	}
	if v.stream != "" {
		// Partial lines are rendered in their arrival position already. Completion
		// only needs to expose the terminal state; do not flush by source, since
		// doing so can reorder fragments that arrived on different streams.
		v.agentPending = make(map[string][]byte)
		v.agentPendingLines = make(map[string]*agentLine)
		v.pendingOrder = nil
		v.agent = appendBounded(v.agent, "["+state+"]")
	}
	return v.renderLocked()
}

func (v *View) AppendAgent(source string, data []byte) error {
	if v == nil || len(data) == 0 {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return v.err
	}
	if source != "stderr" {
		source = "stdout"
	}
	if v.agentPending == nil {
		v.agentPending = make(map[string][]byte)
	}
	if v.agentPendingLines == nil {
		v.agentPendingLines = make(map[string]*agentLine)
	}
	line := v.agentPendingLines[source]
	if line == nil {
		line = &agentLine{source: source}
		v.agentRecords = append(v.agentRecords, line)
		v.agentPendingLines[source] = line
		present := false
		for _, pendingSource := range v.pendingOrder {
			if pendingSource == source {
				present = true
				break
			}
		}
		if !present {
			v.pendingOrder = append(v.pendingOrder, source)
		}
	}
	// Keep an unterminated logical line bounded too. A process can emit an
	// arbitrarily long line (or binary data without newlines), so cap the
	// retained suffix before it enters pane state.
	if len(data) > maxPendingBytes {
		data = data[len(data)-maxPendingBytes:]
	}
	pending := line.data
	if excess := len(pending) + len(data) - maxPendingBytes; excess > 0 {
		pending = pending[excess:]
	}
	line.data = append(pending, data...)
	for {
		complete, rest, found := bytes.Cut(line.data, []byte{'\n'})
		if !found {
			break
		}
		line.data = complete
		line.complete = true
		delete(v.agentPendingLines, source)
		delete(v.agentPending, source)
		if len(rest) == 0 {
			break
		}
		line = &agentLine{source: source, data: rest}
		v.agentRecords = append(v.agentRecords, line)
		v.agentPendingLines[source] = line
	}
	if pendingLine := v.agentPendingLines[source]; pendingLine != nil {
		v.agentPending[source] = append([]byte(nil), pendingLine.data...)
	}
	v.rebuildAgentLocked()
	return v.renderLocked()
}

func (v *View) rebuildAgentLocked() {
	if len(v.agentRecords) > maxLines {
		v.agentRecords = append([]*agentLine(nil), v.agentRecords[len(v.agentRecords)-maxLines:]...)
	}
	v.agentPending = make(map[string][]byte)
	v.agentPendingLines = make(map[string]*agentLine)
	v.pendingOrder = nil
	for _, line := range v.agentRecords {
		if previous := v.agentPendingLines[line.source]; previous == line {
			continue
		}
		// A source's pending record is the last incomplete record for it. A
		// completed record has no continuation until a later fragment arrives.
		if !line.complete {
			v.agentPendingLines[line.source] = line
			v.agentPending[line.source] = append([]byte(nil), line.data...)
			v.pendingOrder = append(v.pendingOrder, line.source)
		}
	}
	v.agent = []string{"[" + v.stream + "]"}
	for _, line := range v.agentRecords {
		prefix := ""
		if line.source == "stderr" {
			prefix = "[stderr] "
		}
		v.agent = append(v.agent, prefix+Sanitize(line.data))
	}
}

// WriteTransient serializes the open-state check and fallback rendering with
// Toggle/rendering. fallback runs only while the primary screen is active, so
// it can clear a transient frame in the same buffer where it was drawn. It
// returns true when the view consumed the liveness frame.
func (v *View) WriteTransient(fallback func() error) (bool, error) {
	if v == nil {
		return false, fallback()
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.err != nil {
		return false, v.err
	}
	if v.open {
		return true, nil
	}
	return false, fallback()
}

// SetTransientClearer registers the primary-screen cleanup that must happen
// after this view restores that screen. The callback is invoked without the
// view lock, so it may safely coordinate with the progress renderer.
func (v *View) SetTransientClearer(clearer func() error) {
	if v == nil {
		return
	}
	v.mu.Lock()
	v.transientClearer = clearer
	v.mu.Unlock()
}

func (v *View) Toggle() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	if v.err != nil {
		v.mu.Unlock()
		return v.err
	}
	v.open = !v.open
	if v.open {
		if err := v.writeLocked("\x1b[?1049h\x1b[?25l"); err != nil {
			v.mu.Unlock()
			return err
		}
		err := v.renderLocked()
		v.mu.Unlock()
		return err
	}
	err := v.writeLocked("\x1b[?25h\x1b[?1049l")
	clearer := v.transientClearer
	v.mu.Unlock()
	if err != nil || clearer == nil {
		return err
	}
	return clearer()
}

func (v *View) scroll(delta int) {
	v.mu.Lock()
	if !v.open {
		v.mu.Unlock()
		return
	}
	if v.focus == 0 {
		v.workflowOffset = max(0, v.workflowOffset+delta)
	} else {
		v.agentOffset = max(0, v.agentOffset+delta)
	}
	_ = v.renderLocked()
	v.mu.Unlock()
}

func (v *View) tail() {
	v.mu.Lock()
	if !v.open {
		v.mu.Unlock()
		return
	}
	if v.focus == 0 {
		v.workflowOffset = 0
	} else {
		v.agentOffset = 0
	}
	_ = v.renderLocked()
	v.mu.Unlock()
}

func (v *View) nextFocus() {
	v.mu.Lock()
	if !v.open {
		v.mu.Unlock()
		return
	}
	v.focus = 1 - v.focus
	_ = v.renderLocked()
	v.mu.Unlock()
}

func (v *View) readKeys(stop <-chan struct{}, reader keyReader, done chan<- struct{}) {
	defer close(done)
	buf := make([]byte, 32)
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, err := reader.Read(buf)
		select {
		case <-stop:
			return
		default:
		}
		if n > 0 {
			v.handleKeys(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func (v *View) handleKeys(keys []byte) {
	v.pending = append(v.pending, keys...)
	i := 0
	for i < len(v.pending) {
		switch v.pending[i] {
		case 'i':
			_ = v.Toggle()
			i++
		case 0x03:
			if v.Interrupt != nil {
				v.Interrupt()
			}
			i++
		case '\t':
			if v.Active() {
				v.nextFocus()
			}
			i++
		case 0x1b:
			if !v.Active() {
				// Navigation escape sequences are view-local. Do not retain a
				// partial sequence while the view is closed.
				i++
				continue
			}
			if i+2 >= len(v.pending) {
				v.pending = append([]byte(nil), v.pending[i:]...)
				return
			}
			if v.pending[i+1] != '[' {
				i++
				continue
			}
			switch v.pending[i+2] {
			case 'A':
				v.scroll(1)
				i += 2
			case 'B':
				v.scroll(-1)
				i += 2
			case 'F':
				v.tail()
				i += 2
			case '5':
				if i+3 >= len(v.pending) {
					v.pending = append([]byte(nil), v.pending[i:]...)
					return
				}
				if v.pending[i+3] == '~' {
					v.scroll(10)
					i += 3
				}
			case '6':
				if i+3 >= len(v.pending) {
					v.pending = append([]byte(nil), v.pending[i:]...)
					return
				}
				if v.pending[i+3] == '~' {
					v.scroll(-10)
					i += 3
				}
			}
			i++
		default:
			i++
		}
	}
	v.pending = v.pending[:0]
}

func (v *View) watchResize(stop <-chan struct{}) {
	resizes, cancel := resizeSignals()
	defer cancel()
	for {
		select {
		case <-stop:
			return
		case <-resizes:
			v.mu.Lock()
			_ = v.renderLocked()
			v.mu.Unlock()
		}
	}
}

// Resize lets tests and alternate runtimes request an immediate reflow.
func (v *View) Resize() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.renderLocked()
}

func (v *View) renderLocked() error {
	if !v.open {
		return v.err
	}
	width, height := 80, 24
	if v.In != nil {
		if w, h, err := term.GetSize(int(v.In.Fd())); err == nil && w > 0 && h > 3 {
			width, height = w, h
		}
	}
	// Two headers and one footer consume three terminal rows.
	body := height - 3
	top := body / 2
	bottom := body - top
	lines := make([]string, 0, height)
	lines = append(lines, "Workflow log")
	lines = append(lines, tailWrapped(v.workflow, width, top, v.workflowOffset)...)
	for len(lines) < top+1 {
		lines = append(lines, "")
	}
	lines = append(lines, "Agent output")
	if len(v.agent) == 0 {
		lines = append(lines, "No active agent output")
	} else {
		lines = append(lines, tailWrapped(v.agent, width, bottom, v.agentOffset)...)
	}
	for len(lines) < height-1 {
		lines = append(lines, "")
	}
	focus := "workflow"
	if v.focus == 1 {
		focus = "agent"
	}
	lines = append(lines, "i: close  Tab/arrows/Page/End: pane navigation ("+focus+")")
	return v.writeLocked("\x1b[H\x1b[2J" + strings.Join(lines, "\n"))
}

func (v *View) writeLocked(value string) error {
	if v.err != nil {
		return v.err
	}
	if _, err := io.WriteString(v.Out, value); err != nil {
		v.err = err
		// A writer can fail after partially accepting an alternate-screen
		// transition. Attempt restoration immediately and leave open set so Stop
		// retries the screen restoration during workflow teardown.
		_, _ = io.WriteString(v.Out, "\x1b[?25h\x1b[?1049l")
		if v.restore != nil && v.In != nil {
			_ = term.Restore(int(v.In.Fd()), v.restore)
			v.restore = nil
		}
		return err
	}
	return nil
}

func (v *View) Err() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.err
}

func appendBounded(lines []string, values ...string) []string {
	lines = append(lines, values...)
	if len(lines) > maxLines {
		lines = append([]string(nil), lines[len(lines)-maxLines:]...)
	}
	return lines
}

func splitLines(value string) []string { return strings.Split(strings.TrimSuffix(value, "\n"), "\n") }

func tailWrapped(lines []string, width, limit, offset int) []string {
	var out []string
	for _, line := range lines {
		runes := []rune(line)
		for len(runes) > width {
			out = append(out, string(runes[:width]))
			runes = runes[width:]
		}
		out = append(out, string(runes))
	}
	end := len(out) - offset
	if end < 0 {
		end = 0
	}
	start := end - limit
	if start < 0 {
		start = 0
	}
	return out[start:end]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Sanitize removes terminal controls while preserving text, tabs and lines.
func Sanitize(data []byte) string {
	var out strings.Builder
	for i := 0; i < len(data); {
		if data[i] == 0x1b {
			i++
			if i < len(data) && data[i] == '[' {
				i++
				for i < len(data) && (data[i] < '@' || data[i] > '~') {
					i++
				}
				if i < len(data) {
					i++
				}
			}
			continue
		}
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			out.WriteRune(utf8.RuneError)
			i++
			continue
		}
		i += size
		if r == '\n' || r == '\t' || (r >= 0x20 && (r < 0x7f || r > 0x9f)) {
			out.WriteRune(r)
		}
	}
	return out.String()
}
