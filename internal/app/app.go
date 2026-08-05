package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dapi/code-converge/internal/codex"
	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/event"
	"github.com/dapi/code-converge/internal/repository"
	"github.com/dapi/code-converge/internal/runner"
	"github.com/dapi/code-converge/internal/session"
	"github.com/dapi/code-converge/internal/terminal"
	selfupdate "github.com/dapi/code-converge/internal/update"
	"github.com/dapi/code-converge/internal/version"
	"github.com/dapi/code-converge/internal/workflow"
	"golang.org/x/term"
)

type optionalFlag struct{ target *config.OptionalString }

type globalFlagSpec struct {
	name, group, description string
	bind                     func(*flag.FlagSet, *config.Overrides)
}

var globalFlagSpecs = []globalFlagSpec{
	{"log-format", "Output", "Workflow output format: human or kv.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "log-format", &o.LogFormat) }},
	{"heartbeat", "Output", "Human-output liveness interval.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "heartbeat", &o.Heartbeat) }},
	{"color", "Output", "Interactive human-output color: auto, always, or never.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "color", &o.Color) }},
	{"mode", "Workflow", "Execution profile: fast or best.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "mode", &o.Mode) }},
	{"max-cycles", "Workflow", "Maximum fix-findings attempts per review phase.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "max-cycles", &o.MaxCycles) }},
	{"max-ci-recoveries", "Workflow", "Maximum CI recovery attempts.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "max-ci-recoveries", &o.MaxCIRecoveries) }},
	{"ci-timeout", "Workflow", "Maximum time to wait for applicable CI (default 60m).", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "ci-timeout", &o.CITimeout) }},
	{"review-model", "Stage overrides", "Review model.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "review-model", &o.ReviewModel) }},
	{"review-reasoning-effort", "Stage overrides", "Review reasoning effort.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "review-reasoning-effort", &o.ReviewEffort) }},
	{"fix-model", "Stage overrides", "Fix-findings model.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "fix-model", &o.FixModel) }},
	{"fix-reasoning-effort", "Stage overrides", "Fix-findings reasoning effort.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "fix-reasoning-effort", &o.FixEffort) }},
	{"fix-prompt-file", "Stage overrides", "Fix-findings prompt file.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "fix-prompt-file", &o.FixPromptPath) }},
	{"review-prompt-file", "Stage overrides", "Explicit Markdown review prompt file.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "review-prompt-file", &o.ReviewPromptPath) }},
	{"review-prompt", "Stage overrides", "Project-local review prompt name.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "review-prompt", &o.ReviewPromptName) }},
	{"document-review", "Stage overrides", "Review changed Markdown documentation.", func(f *flag.FlagSet, o *config.Overrides) {
		f.BoolVar(&o.DocumentReview, "document-review", false, "review changed Markdown documentation")
	}},
	{"document-fix-prompt-file", "Stage overrides", "Markdown fix prompt for document review.", func(f *flag.FlagSet, o *config.Overrides) {
		bind(f, "document-fix-prompt-file", &o.DocumentFixPromptPath)
	}},
	{"ci-fix-model", "Stage overrides", "CI-fix model.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "ci-fix-model", &o.CIFixModel) }},
	{"ci-fix-reasoning-effort", "Stage overrides", "CI-fix reasoning effort.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "ci-fix-reasoning-effort", &o.CIFixEffort) }},
	{"ci-fix-prompt-file", "Stage overrides", "CI-fix prompt file.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "ci-fix-prompt-file", &o.CIFixPromptPath) }},
	{"review-base", "Workflow", "Review base override.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "review-base", &o.ReviewBase) }},
	{"session-log-dir", "Diagnostics", "Diagnostic session-log directory.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "session-log-dir", &o.SessionLogDir) }},
	{"session-log-retention", "Diagnostics", "Diagnostic session-log retention.", func(f *flag.FlagSet, o *config.Overrides) { bind(f, "session-log-retention", &o.SessionLogRetention) }},
	{"no-session-log", "Diagnostics", "Disable diagnostic session logging.", func(f *flag.FlagSet, o *config.Overrides) {
		f.BoolVar(&o.NoSessionLog, "no-session-log", false, "disable diagnostic session logging")
	}},
}

func (f optionalFlag) String() string {
	if f.target == nil {
		return ""
	}
	return f.target.Value
}

func (f optionalFlag) Set(value string) error {
	f.target.Value, f.target.Set = value, true
	return nil
}

type App struct {
	Stdout        io.Writer
	Stderr        io.Writer
	Stdin         *os.File
	Cwd           string
	Home          string
	Runner        runner.Runner
	Now           func() time.Time
	IsTerminal    func(io.Writer) bool
	TerminalWidth func(io.Writer) (int, error)
	LookupEnv     func(string) (string, bool)
	Updater       selfupdate.Runner
}

func (a App) Run(ctx context.Context, args []string) int {
	stdout, stderr := a.Stdout, a.Stderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if helpCommand(stdout, args) {
		return workflow.ExitSuccess
	}
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintf(stdout, "code-converge v%s\n", version.Version)
		return workflow.ExitSuccess
	}
	if len(args) > 0 && args[0] == "update" {
		assumeYes, err := updateArgs(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "code-converge update: %v\n", err)
			return workflow.ExitOperational
		}
		updater := a.Updater
		if updater == nil {
			updater = selfupdate.Service{Version: version.Version, Stdout: stdout, Stderr: stderr}
		}
		return updater.Run(ctx, assumeYes)
	}
	if len(args) > 0 && args[0] == "init-document-review-prompt" {
		force, err := initDocumentReviewArgs(args[1:])
		if err != nil {
			fmt.Fprintf(stderr, "code-converge init-document-review-prompt: %v\n", err)
			return workflow.ExitOperational
		}
		cwd := a.Cwd
		if cwd == "" {
			cwd, err = os.Getwd()
			if err != nil {
				fmt.Fprintf(stderr, "code-converge: current directory: %v\n", err)
				return workflow.ExitOperational
			}
		}
		root, err := config.FindGitRoot(cwd)
		if err != nil {
			fmt.Fprintf(stderr, "code-converge: %v\n", err)
			return workflow.ExitOperational
		}
		path := filepath.Join(root, ".code-converge", "default.md")
		if info, err := os.Lstat(path); err == nil {
			if !info.Mode().IsRegular() {
				fmt.Fprintf(stderr, "code-converge init-document-review-prompt: %s is not a regular file\n", path)
				return workflow.ExitOperational
			}
			if !force {
				fmt.Fprintf(stderr, "code-converge init-document-review-prompt: %s already exists; use --force to overwrite\n", path)
				return workflow.ExitOperational
			}
		} else if !os.IsNotExist(err) {
			fmt.Fprintf(stderr, "code-converge init-document-review-prompt: %v\n", err)
			return workflow.ExitOperational
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			fmt.Fprintf(stderr, "code-converge init-document-review-prompt: %v\n", err)
			return workflow.ExitOperational
		}
		if err := writeDocumentReviewPrompt(path, []byte(config.DocumentReviewPrompt+"\n"), force); err != nil {
			fmt.Fprintf(stderr, "code-converge init-document-review-prompt: %v\n", err)
			return workflow.ExitOperational
		}
		fmt.Fprintln(stdout, path)
		return workflow.ExitSuccess
	}
	cwd := a.Cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "code-converge: current directory: %v\n", err)
			return workflow.ExitOperational
		}
	}

	overrides := config.Overrides{}
	configCommand := len(args) > 0 && args[0] == "config"
	flags := flag.NewFlagSet("code-converge", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	for _, spec := range globalFlagSpecs {
		spec.bind(flags, &overrides)
	}

	if len(args) > 0 && args[0] == "config" {
		args = append(append([]string{}, args[1:]...), "config")
	}
	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(stderr, "code-converge: %v\n", err)
		if !configCommand {
			logger := event.Logger{Out: stdout, Err: stderr, Now: a.Now, Format: "human"}
			started := time.Now()
			_ = logger.Emit("run_started")
			_ = logger.Emit("run_completed", event.F("status", "operational_failure"), event.F("exit_code", "2"), event.F("total_duration_ms", fmt.Sprint(time.Since(started).Milliseconds())))
		}
		return workflow.ExitOperational
	}
	if flags.NArg() > 1 || (flags.NArg() == 1 && flags.Arg(0) != "config") {
		fmt.Fprintln(stderr, "code-converge: usage: code-converge [flags] [config]")
		return workflow.ExitOperational
	}

	startupFormat := "human"
	if resolved, resolveErr := config.ResolveLogFormat(cwd, a.Home, overrides.LogFormat); resolveErr == nil {
		startupFormat = resolved
	}
	cfg, err := config.Load(cwd, a.Home, overrides)
	if err != nil {
		if flags.NArg() == 1 {
			fmt.Fprintf(stderr, "code-converge: configuration: %v\n", err)
			return workflow.ExitOperational
		}
		logger := event.Logger{Out: stdout, Err: stderr, Now: a.Now, Format: startupFormat}
		started := time.Now()
		_ = logger.Emit("run_started")
		fmt.Fprintf(stderr, "code-converge: configuration: %v\n", err)
		_ = logger.Emit("run_completed", event.F("status", "operational_failure"), event.F("exit_code", "2"), event.F("total_duration_ms", fmt.Sprint(time.Since(started).Milliseconds())))
		return workflow.ExitOperational
	}
	if flags.NArg() == 1 {
		_, _ = io.WriteString(stdout, config.Format(cfg))
		return workflow.ExitSuccess
	}

	processRunner := a.Runner
	if processRunner == nil {
		processRunner = runner.Exec{Executable: "codex", Dir: cwd}
	}
	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	var view *terminal.View
	stdin := a.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	lookup := a.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	termName, _ := lookup("TERM")
	interactiveTerminal := cfg.LogFormat == "human" && a.isTerminal(stdout) && termName != "" && termName != "dumb"
	if interactiveTerminal && terminal.IsTerminalWriter(stdout) {
		candidate := terminal.New(stdout, stdin)
		candidate.Interrupt = cancelRun
		if candidate.Eligible() && candidate.Start() == nil {
			view = candidate
			defer view.Stop()
		}
	}
	logger := event.Logger{
		Out: stdout, Err: stderr, Now: a.Now, Format: cfg.LogFormat, Heartbeat: cfg.Heartbeat,
		// A TTY alone is insufficient: TERM=dumb (and an absent TERM) does not
		// promise to interpret carriage-return/ANSI transient frames. Keep all
		// output permanent in that environment so captured session output cannot
		// be corrupted by terminal control sequences.
		Interactive: interactiveTerminal, ColorDepth: a.colorDepth(cfg, stdout), View: view,
	}
	if err := logger.InteractiveHint(); err != nil {
		fmt.Fprintf(stderr, "code-converge: interactive hint: %v\n", err)
		return workflow.ExitOperational
	}
	if logger.Interactive && cfg.LogFormat == "human" && cfg.Heartbeat == 0 {
		logger.TerminalWidth = func() (int, error) { return a.terminalWidth(stdout) }
	}
	if !cfg.NoSessionLog {
		writer, sessionErr := session.Start(session.Config{
			Dir: cfg.SessionLogDir, Retention: cfg.SessionLogRetention, Now: a.Now,
			Diagnostic: logger.Diagnostic,
		})
		if sessionErr != nil {
			logger.Diagnostic("session log", sessionErr)
		}
		if writer != nil {
			defer writer.Close()
			processRunner = session.Wrap(processRunner, writer)
			if err := logger.SessionLog(writer.Path()); err != nil {
				logger.Diagnostic("write session log path", err)
			}
		}
	}
	reviewScope := &repository.ReviewScope{Runner: processRunner, Base: cfg.ReviewBase, Root: cfg.Root, DocumentReview: cfg.DocumentReview}
	defer reviewScope.Close()
	var agentOutput func(string, []byte)
	if view != nil {
		agentOutput = logger.AgentOutput
	}
	agent := codex.Adapter{Runner: processRunner, Config: cfg, ReviewScope: reviewScope, Output: agentOutput}
	w := workflow.Workflow{Config: cfg, Agent: agent, Repository: repository.Status{Runner: processRunner}, Log: &logger, Err: stderr, Now: a.Now}
	return w.Run(runCtx)
}

func rootUsage(out io.Writer) {
	fmt.Fprintln(out, "usage: code-converge [flags] [config]")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  config                 Show effective configuration and its sources.")
	fmt.Fprintln(out, "  update [--yes|-y]      Check for and install a newer release.")
	fmt.Fprintln(out, "  init-document-review-prompt [--force]  Write the editable document-review prompt.")
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "Global options:")
	group := ""
	for _, spec := range globalFlagSpecs {
		if spec.group != group {
			group = spec.group
			fmt.Fprintf(out, "  %s:\n", group)
		}
		fmt.Fprintf(out, "    --%-25s %s\n", spec.name, spec.description)
	}
	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "See README.md for the full configuration reference.")
}

func helpCommand(out io.Writer, args []string) bool {
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		rootUsage(out)
		return true
	}
	if len(args) != 2 || (args[1] != "-h" && args[1] != "--help") {
		return false
	}
	switch args[0] {
	case "config":
		fmt.Fprintln(out, "usage: code-converge config [global options]")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Show effective configuration values and their sources without starting a workflow.")
		return true
	case "update":
		fmt.Fprintln(out, "usage: code-converge update [--yes|-y]")
		fmt.Fprintln(out, "")
		fmt.Fprintln(out, "Check for and install a newer release. --yes and -y skip confirmation.")
		return true
	case "init-document-review-prompt":
		fmt.Fprintln(out, "usage: code-converge init-document-review-prompt [--force]")
		return true
	default:
		return false
	}
}

func initDocumentReviewArgs(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) == 1 && args[0] == "--force" {
		return true, nil
	}
	return false, fmt.Errorf("usage: code-converge init-document-review-prompt [--force]")
}

func updateArgs(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) == 1 && (args[0] == "--yes" || args[0] == "-y") {
		return true, nil
	}
	return false, fmt.Errorf("usage: code-converge update [--yes|-y]")
}

func (a App) isTerminal(out io.Writer) bool {
	if a.IsTerminal != nil {
		return a.IsTerminal(out)
	}
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}

func (a App) terminalWidth(out io.Writer) (int, error) {
	if a.TerminalWidth != nil {
		return a.TerminalWidth(out)
	}
	file, ok := out.(*os.File)
	if !ok {
		return 0, fmt.Errorf("stdout is not a file")
	}
	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil {
		return 0, err
	}
	return width, nil
}

func (a App) colorDepth(cfg config.Config, out io.Writer) int {
	if cfg.Color == "never" || !a.isTerminal(out) {
		return 0
	}
	lookup := a.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}
	if _, disabled := lookup("NO_COLOR"); disabled {
		return 0
	}
	term, _ := lookup("TERM")
	if term == "" || term == "dumb" {
		return 0
	}
	colorTerm, _ := lookup("COLORTERM")
	if strings.Contains(strings.ToLower(colorTerm), "truecolor") || strings.Contains(strings.ToLower(colorTerm), "24bit") {
		return 3
	}
	if strings.Contains(strings.ToLower(term), "256color") {
		return 2
	}
	return 1
}

func bind(flags *flag.FlagSet, name string, target *config.OptionalString) {
	flags.Var(optionalFlag{target: target}, name, strings.ReplaceAll(name, "-", " "))
}
