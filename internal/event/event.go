package event

import (
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-runewidth"
)

var keyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type Field struct {
	Key   string
	Value string
}

func F(key, value string) Field { return Field{Key: key, Value: value} }

type TickFactory func(time.Duration) (<-chan time.Time, func())

type Logger struct {
	Out         io.Writer
	Err         io.Writer
	Now         func() time.Time
	Format      string
	Heartbeat   time.Duration
	Interactive bool
	ColorDepth  int // 0=none, 1=basic, 2=ANSI-256, 3=true color
	Tick        TickFactory
	// TerminalWidth returns the current interactive stdout width in cells. It is
	// used only for transient human liveness frames.
	TerminalWidth func() (int, error)
	// HumanMaxCycles and HumanMaxCIRecoveries provide the configured budgets for
	// operator-facing progress only. They never affect workflow behavior or kv.
	HumanMaxCycles       int
	HumanMaxCIRecoveries int

	mu             sync.Mutex
	transient      bool
	transientCells int
}

// StageContext supplies the facts needed by a liveness line. It deliberately
// mirrors event fields so the permanent and transient human output agree.
type StageContext struct {
	Stage           string
	Model           string
	ReasoningEffort string
	ReviewPhase     int
	Cycle           int
}

func (l *Logger) Emit(eventName string, fields ...Field) error {
	if err := validate(eventName, fields); err != nil {
		return err
	}
	line, err := l.render(eventName, fields)
	if err != nil || line == "" {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.clearLocked(); err != nil {
		return err
	}
	_, err = fmt.Fprintln(l.Out, line)
	return err
}

func (l *Logger) Diagnostic(message string, err error) {
	if l.Err == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.clearLocked()
	fmt.Fprintf(l.Err, "code-converge: %s: %v\n", message, err)
}

type Liveness struct {
	stop chan struct{}
	done chan struct{}
	err  chan error
	once sync.Once
}

func (l *Logger) StartLiveness(ctx context.Context, stage StageContext, started time.Time, cancel context.CancelFunc) *Liveness {
	live := &Liveness{stop: make(chan struct{}), done: make(chan struct{}), err: make(chan error, 1)}
	if l.normalizedFormat() != "human" || (l.Heartbeat == 0 && !l.Interactive) {
		close(live.done)
		return live
	}
	go l.runLiveness(ctx, live, stage, started, cancel)
	return live
}

func (live *Liveness) Stop() error {
	live.once.Do(func() { close(live.stop) })
	<-live.done
	select {
	case err := <-live.err:
		return err
	default:
		return nil
	}
}

func (l *Logger) runLiveness(ctx context.Context, live *Liveness, stage StageContext, started time.Time, cancel context.CancelFunc) {
	defer close(live.done)
	select {
	case <-ctx.Done():
		return
	case <-live.stop:
		return
	default:
	}
	interval := 100 * time.Millisecond
	if l.Heartbeat > 0 {
		interval = l.Heartbeat
	}
	ticks, stopTicker := l.ticker(interval)
	defer stopTicker()
	frame := 0
	if l.Heartbeat == 0 {
		if err := l.writeTransient(stage, l.now().Sub(started), frame); err != nil {
			live.err <- err
			cancel()
			return
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-live.stop:
			return
		case now := <-ticks:
			var err error
			if l.Heartbeat > 0 {
				err = l.writeHeartbeat(stage, now.Sub(started))
			} else {
				frame++
				err = l.writeTransient(stage, now.Sub(started), frame)
			}
			if err != nil {
				live.err <- err
				cancel()
				return
			}
		}
	}
}

func (l *Logger) writeHeartbeat(stage StageContext, elapsed time.Duration) error {
	label := l.livenessLabel(stage, false)
	if label == "" {
		return fmt.Errorf("unknown liveness stage %q", stage.Stage)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := fmt.Fprintf(l.Out, "%s%s still running (%s)\n", l.humanPrefix(l.stageAttempt(stage), stage.Model, stage.ReasoningEffort), label, HumanDuration(elapsed))
	return err
}

func (l *Logger) writeTransient(stage StageContext, elapsed time.Duration, frame int) error {
	label := l.livenessLabel(stage, true)
	if label == "" {
		return fmt.Errorf("unknown liveness stage %q", stage.Stage)
	}
	seconds := int64(elapsed / time.Second)
	if seconds < 0 {
		seconds = 0
	}
	line := fmt.Sprintf("%s%s... %s", l.humanPrefix(l.stageAttempt(stage), stage.Model, stage.ReasoningEffort), label, compoundSeconds(seconds))
	l.mu.Lock()
	defer l.mu.Unlock()
	width, err := l.terminalWidthLocked()
	if err != nil {
		return err
	}
	line, cells := truncateCells(line, width-1)
	if cells == 0 {
		return fmt.Errorf("liveness line does not fit terminal width %d", width)
	}
	if err := l.clearLocked(); err != nil {
		return err
	}
	if l.ColorDepth > 0 {
		line = shimmer(line, frame, l.ColorDepth)
	}
	if _, err := io.WriteString(l.Out, "\r\x1b[2K"+line); err != nil {
		return err
	}
	l.transient = true
	l.transientCells = cells
	return nil
}

func (l *Logger) clearLocked() error {
	if !l.transient {
		return nil
	}
	width, err := l.terminalWidthLocked()
	if err != nil {
		return err
	}
	rows := (l.transientCells + width - 1) / width
	if rows < 1 {
		return fmt.Errorf("invalid transient footprint %d", l.transientCells)
	}
	var output strings.Builder
	output.WriteString("\r\x1b[2K")
	for row := 1; row < rows; row++ {
		output.WriteString("\x1b[1A\r\x1b[2K")
	}
	_, err = io.WriteString(l.Out, output.String())
	if err == nil {
		l.transient = false
		l.transientCells = 0
	}
	return err
}

func (l *Logger) terminalWidthLocked() (int, error) {
	if l.TerminalWidth == nil {
		return 0, fmt.Errorf("interactive terminal width is unavailable")
	}
	width, err := l.TerminalWidth()
	if err != nil {
		return 0, fmt.Errorf("terminal width: %w", err)
	}
	if width < 2 {
		return 0, fmt.Errorf("invalid terminal width %d", width)
	}
	return width, nil
}

func truncateCells(value string, limit int) (string, int) {
	if limit < 1 {
		return "", 0
	}
	width := 0
	end := 0
	for index, r := range value {
		cells := runewidth.RuneWidth(r)
		if cells < 0 || width+cells > limit {
			break
		}
		width += cells
		end = index + len(string(r))
	}
	return value[:end], width
}

func (l *Logger) ticker(interval time.Duration) (<-chan time.Time, func()) {
	if l.Tick != nil {
		return l.Tick(interval)
	}
	ticker := time.NewTicker(interval)
	return ticker.C, ticker.Stop
}

func (l *Logger) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

func (l *Logger) normalizedFormat() string {
	if l.Format == "" {
		return "kv"
	}
	return l.Format
}

func (l *Logger) render(eventName string, fields []Field) (string, error) {
	if l.normalizedFormat() == "human" {
		if l.Interactive && eventName == "stage_started" {
			return "", nil
		}
		line, err := renderHuman(eventName, fields, l.HumanMaxCycles, l.HumanMaxCIRecoveries)
		if line == "" || err != nil {
			return line, err
		}
		return l.humanPrefix(l.eventAttempt(fields), fieldValue(fields, "model"), fieldValue(fields, "reasoning_effort")) + line, nil
	}
	all := append([]Field{{Key: "ts", Value: l.now().UTC().Format(time.RFC3339)}, {Key: "event", Value: eventName}}, fields...)
	parts := make([]string, 0, len(all))
	for _, field := range all {
		parts = append(parts, field.Key+"="+field.Value)
	}
	return strings.Join(parts, " "), nil
}

func renderHuman(eventName string, fields []Field, maxCycles, maxCIRecoveries int) (string, error) {
	values := make(map[string]string, len(fields))
	for _, field := range fields {
		values[field.Key] = field.Value
	}
	duration := func(key string) (string, error) {
		ms, err := strconv.ParseInt(values[key], 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid %s %q", key, values[key])
		}
		return HumanDuration(time.Duration(ms) * time.Millisecond), nil
	}
	switch eventName {
	case "run_started":
		return "", nil
	case "stage_started":
		switch values["stage"] {
		case "review":
			if values["review_phase"] != "" && values["review_phase"] != "1" {
				phase, err := strconv.Atoi(values["review_phase"])
				if err != nil || phase < 2 {
					return "", fmt.Errorf("invalid review_phase %q", values["review_phase"])
				}
				return fmt.Sprintf("Review started (phase %d after CI recovery %d)", phase, phase-1), nil
			}
			return "Review started", nil
		case "fix-findings":
			return "Fixing findings", nil
		case "finalize":
			return "Finalizing", nil
		case "fix-ci":
			return "CI recovery", nil
		}
	case "review_completed":
		d, err := duration("duration_ms")
		if err != nil {
			return "", err
		}
		prefix := "Review"
		switch values["status"] {
		case "clean":
			return fmt.Sprintf("%s: clean (%s)", prefix, d), nil
		case "failed":
			return fmt.Sprintf("%s failed (%s)", prefix, d), nil
		case "findings":
			parts := make([]string, 0, 5)
			for _, severity := range []string{"critical", "high", "medium", "low", "unknown"} {
				if count := values["findings_"+severity]; count != "" && count != "0" {
					parts = append(parts, count+" "+severity)
				}
			}
			if len(parts) == 0 {
				return "", fmt.Errorf("findings status has no non-zero severity counts")
			}
			total := values["findings_total"]
			noun := "findings"
			if total == "1" {
				noun = "finding"
			}
			return fmt.Sprintf("%s: %s %s — %s (%s)", prefix, total, noun, strings.Join(parts, ", "), d), nil
		}
	case "stage_completed":
		d, err := duration("duration_ms")
		if err != nil {
			return "", err
		}
		switch values["stage"] {
		case "fix-findings":
			switch values["status"] {
			case "success":
				return fmt.Sprintf("Findings fixed (%s)", d), nil
			case "failed":
				return fmt.Sprintf("Fixing findings failed (%s)", d), nil
			}
		case "fix-ci":
			switch values["status"] {
			case "success":
				return fmt.Sprintf("CI recovery fixed (%s)", d), nil
			case "failed":
				return fmt.Sprintf("CI recovery failed (%s)", d), nil
			}
		case "finalize":
			switch values["verdict"] {
			case "SUCCESS":
				return fmt.Sprintf("Finalized successfully (%s)", d), nil
			case "CI_FAILED":
				return fmt.Sprintf("Finalized, but CI is failing (%s)", d), nil
			case "FAILED":
				return fmt.Sprintf("Finalization failed (%s)", d), nil
			case "":
				if values["status"] == "failed" {
					return fmt.Sprintf("Finalization failed (%s)", d), nil
				}
			}
		}
	case "step_completed":
		labels := map[string]string{"commit": "Commit", "push": "Push", "change_request": "Change request", "ci": "CI"}
		statuses := map[string]string{"success": "done", "skipped": "not needed", "failed": "failed", "unknown": "unknown"}
		label, labelOK := labels[values["step"]]
		status, statusOK := statuses[values["status"]]
		if !labelOK || !statusOK {
			break
		}
		return fmt.Sprintf("  %s: %s", label, status), nil
	case "run_completed":
		d, err := duration("total_duration_ms")
		if err != nil {
			return "", err
		}
		switch values["status"] {
		case "success":
			return fmt.Sprintf("Done (%s)", d), nil
		case "findings_remaining":
			return fmt.Sprintf("Stopped: review findings remain (%s, exit 1)", d), nil
		case "operational_failure":
			return fmt.Sprintf("Failed due to an operational error (%s, exit 2)", d), nil
		case "ci_failure":
			return fmt.Sprintf("Stopped: CI is still failing (%s, exit 3)", d), nil
		}
	}
	return "", fmt.Errorf("unsupported human event %s with fields %#v", eventName, fields)
}

func (l *Logger) humanPrefix(attempt, model, effort string) string {
	prefix := l.now().Format("15:04:05") + " "
	if attempt != "" {
		prefix += "[" + attempt + "] "
	}
	if model != "" && effort != "" {
		prefix += "[" + model + "/" + effort + "] "
	}
	return prefix
}

func (l *Logger) eventAttempt(fields []Field) string {
	stage := fieldValue(fields, "stage")
	switch stage {
	case "review", "fix-findings":
		if cycle := fieldValue(fields, "cycle"); cycle != "" {
			return fmt.Sprintf("%s/%d", cycle, l.HumanMaxCycles)
		}
	case "fix-ci":
		if phase := fieldValue(fields, "review_phase"); phase != "" {
			return fmt.Sprintf("%s/%d", phase, l.HumanMaxCIRecoveries)
		}
	}
	return ""
}

func (l *Logger) livenessLabel(stage StageContext, transient bool) string {
	labels := map[string][2]string{
		"review":       {"Reviewing", "Review"},
		"fix-findings": {"Fixing findings", "Fixing findings"},
		"finalize":     {"Finalizing", "Finalization"},
		"fix-ci":       {"CI recovery", "CI recovery"},
	}
	label, ok := labels[stage.Stage]
	if !ok {
		return ""
	}
	if transient {
		return label[0]
	}
	return label[1]
}

func (l *Logger) stageAttempt(stage StageContext) string {
	switch stage.Stage {
	case "review", "fix-findings":
		return fmt.Sprintf("%d/%d", stage.Cycle, l.HumanMaxCycles)
	case "fix-ci":
		return fmt.Sprintf("%d/%d", stage.ReviewPhase, l.HumanMaxCIRecoveries)
	default:
		return ""
	}
}

func fieldValue(fields []Field, key string) string {
	for _, field := range fields {
		if field.Key == key {
			return field.Value
		}
	}
	return ""
}

func validate(eventName string, fields []Field) error {
	if eventName == "" || strings.ContainsAny(eventName, " \t\r\n=") {
		return fmt.Errorf("invalid event name %q", eventName)
	}
	for _, field := range fields {
		if !keyPattern.MatchString(field.Key) {
			return fmt.Errorf("invalid event key %q", field.Key)
		}
		if field.Value == "" || strings.ContainsAny(field.Value, " \t\r\n=") {
			return fmt.Errorf("invalid event value for %s", field.Key)
		}
	}
	return nil
}

func HumanDuration(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	if value < time.Minute {
		seconds := math.Round(value.Seconds()*10) / 10
		return strings.TrimSuffix(strconv.FormatFloat(seconds, 'f', 1, 64), ".0") + "s"
	}
	return compoundSeconds(int64(math.Round(value.Seconds())))
}

func compoundSeconds(total int64) string {
	if total < 0 {
		total = 0
	}
	hours, remainder := total/3600, total%3600
	minutes, seconds := remainder/60, remainder%60
	parts := make([]string, 0, 3)
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	return strings.Join(parts, " ")
}

func shimmer(text string, frame, depth int) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}
	center := shimmerHighlightCenter(len(runes), frame)
	var out strings.Builder
	for i, r := range runes {
		distance := i - center
		if distance < 0 {
			distance = -distance
		}
		rgb := shimmerColor(distance)
		switch depth {
		case 3:
			fmt.Fprintf(&out, "\x1b[38;2;%d;%d;%dm%c", rgb[0], rgb[1], rgb[2], r)
		case 2:
			index := 16 + 36*(rgb[0]/51) + 6*(rgb[1]/51) + rgb[2]/51
			fmt.Fprintf(&out, "\x1b[38;5;%dm%c", index, r)
		default:
			code := 35
			if distance < 2 {
				code = 36
			}
			fmt.Fprintf(&out, "\x1b[%dm%c", code, r)
		}
	}
	out.WriteString("\x1b[0m")
	return out.String()
}

func shimmerHighlightCenter(length, frame int) int {
	travel := length + 8
	cycle := travel * 2
	center := (frame / 2) % cycle
	if center >= travel {
		center = cycle - center
	}
	return center - 4
}

func shimmerColor(distance int) [3]int {
	switch distance {
	case 0:
		return [3]int{255, 255, 255}
	case 1:
		return [3]int{234, 218, 255}
	case 2:
		return [3]int{207, 181, 250}
	case 3:
		return [3]int{177, 143, 230}
	default:
		return [3]int{147, 112, 196}
	}
}
