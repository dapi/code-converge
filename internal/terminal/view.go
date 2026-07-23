// Package terminal owns the narrow interactive terminal surface used by the
// human progress renderer. It deliberately has no workflow knowledge.
package terminal

import (
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"golang.org/x/term"
)

const maxLines = 2000

// View stores the two independent pane histories and, while open, renders
// them in the terminal's alternate screen.
type View struct {
	Out io.Writer
	In  interface {
		Fd() uintptr
		Read([]byte) (int, error)
	}

	mu             sync.Mutex
	open           bool
	workflow       []string
	agent          []string
	stream         string
	focus          int
	workflowOffset int
	agentOffset    int
	pending        []byte
	restore        *term.State
	stop           chan struct{}
	done           chan struct{}
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
	return v != nil && v.In != nil && term.IsTerminal(int(v.In.Fd()))
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
	v.mu.Unlock()
	go v.readKeys(v.stop)
	return nil
}

// Stop closes the split view and restores stdin. A raw terminal read cannot be
// cancelled portably, so its reader exits on the next input after restoration.
func (v *View) Stop() {
	if v == nil {
		return
	}
	v.mu.Lock()
	stop, done, state := v.stop, v.done, v.restore
	v.stop, v.done, v.restore = nil, nil, nil
	wasOpen := v.open
	v.open = false
	v.mu.Unlock()
	if stop != nil {
		close(stop)
	}
	if wasOpen {
		_, _ = io.WriteString(v.Out, "\x1b[?25h\x1b[?1049l")
	}
	if state != nil {
		_ = term.Restore(int(v.In.Fd()), state)
	}
	// A raw terminal read cannot be cancelled portably. Restoring the terminal
	// makes the reader return on the next input; do not block workflow teardown
	// waiting for that operator action.
	_ = done
}

func (v *View) Active() bool {
	if v == nil {
		return false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.open
}

func (v *View) AppendWorkflow(line string) {
	if v == nil || line == "" {
		return
	}
	v.mu.Lock()
	v.workflow = appendBounded(v.workflow, splitLines(line)...)
	v.renderLocked()
	v.mu.Unlock()
}

func (v *View) StartAgent(identity string) {
	if v == nil {
		return
	}
	v.mu.Lock()
	v.stream = identity
	v.agent = []string{"[" + identity + "]"}
	v.renderLocked()
	v.mu.Unlock()
}

func (v *View) CompleteAgent(state string) {
	if v == nil {
		return
	}
	v.mu.Lock()
	if v.stream != "" {
		v.agent = appendBounded(v.agent, "["+state+"]")
	}
	v.renderLocked()
	v.mu.Unlock()
}

func (v *View) AppendAgent(source string, data []byte) {
	if v == nil || len(data) == 0 {
		return
	}
	prefix := ""
	if source == "stderr" {
		prefix = "[stderr] "
	}
	v.mu.Lock()
	v.agent = appendBounded(v.agent, splitLines(prefix+Sanitize(data))...)
	v.renderLocked()
	v.mu.Unlock()
}

func (v *View) Toggle() {
	if v == nil {
		return
	}
	v.mu.Lock()
	v.open = !v.open
	if v.open {
		_, _ = io.WriteString(v.Out, "\x1b[?1049h\x1b[?25l")
		v.renderLocked()
	} else {
		_, _ = io.WriteString(v.Out, "\x1b[?25h\x1b[?1049l")
	}
	v.mu.Unlock()
}

func (v *View) scroll(delta int) {
	v.mu.Lock()
	if v.focus == 0 {
		v.workflowOffset = max(0, v.workflowOffset+delta)
	} else {
		v.agentOffset = max(0, v.agentOffset+delta)
	}
	v.renderLocked()
	v.mu.Unlock()
}

func (v *View) tail() {
	v.mu.Lock()
	if v.focus == 0 {
		v.workflowOffset = 0
	} else {
		v.agentOffset = 0
	}
	v.renderLocked()
	v.mu.Unlock()
}

func (v *View) nextFocus() {
	v.mu.Lock()
	v.focus = 1 - v.focus
	v.renderLocked()
	v.mu.Unlock()
}

func (v *View) readKeys(stop <-chan struct{}) {
	defer func() {
		v.mu.Lock()
		if v.done != nil {
			close(v.done)
		}
		v.mu.Unlock()
	}()
	buf := make([]byte, 32)
	for {
		select {
		case <-stop:
			return
		default:
		}
		n, err := v.In.Read(buf)
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
			v.Toggle()
			i++
		case '\t':
			v.nextFocus()
			i++
		case 0x1b:
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

func (v *View) renderLocked() {
	if !v.open {
		return
	}
	width, height := 80, 24
	if v.In != nil {
		if w, h, err := term.GetSize(int(v.In.Fd())); err == nil && w > 0 && h > 3 {
			width, height = w, h
		}
	}
	top := (height - 2) / 2
	bottom := height - 2 - top
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
	_, _ = io.WriteString(v.Out, "\x1b[H\x1b[2J"+strings.Join(lines, "\n"))
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
