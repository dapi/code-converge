package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dapi/code-converge/internal/codex"
	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/event"
	"github.com/dapi/code-converge/internal/runner"
	"github.com/dapi/code-converge/internal/version"
	"github.com/dapi/code-converge/internal/workflow"
)

type optionalFlag struct{ target *config.OptionalString }

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
	Stdout     io.Writer
	Stderr     io.Writer
	Cwd        string
	Home       string
	Runner     runner.Runner
	Now        func() time.Time
	IsTerminal func(io.Writer) bool
	LookupEnv  func(string) (string, bool)
}

func (a App) Run(ctx context.Context, args []string) int {
	stdout, stderr := a.Stdout, a.Stderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	if len(args) == 1 && args[0] == "--version" {
		fmt.Fprintf(stdout, "code-converge v%s\n", version.Version)
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
	bind(flags, "log-format", &overrides.LogFormat)
	bind(flags, "heartbeat", &overrides.Heartbeat)
	bind(flags, "color", &overrides.Color)
	bind(flags, "mode", &overrides.Mode)
	bind(flags, "max-cycles", &overrides.MaxCycles)
	bind(flags, "max-ci-recoveries", &overrides.MaxCIRecoveries)
	bind(flags, "review-model", &overrides.ReviewModel)
	bind(flags, "review-reasoning-effort", &overrides.ReviewEffort)
	bind(flags, "fix-model", &overrides.FixModel)
	bind(flags, "fix-reasoning-effort", &overrides.FixEffort)
	bind(flags, "fix-prompt-file", &overrides.FixPromptPath)
	bind(flags, "finalize-model", &overrides.FinalizeModel)
	bind(flags, "finalize-reasoning-effort", &overrides.FinalizeEffort)
	bind(flags, "finalize-prompt-file", &overrides.FinalizePromptPath)
	bind(flags, "ci-fix-model", &overrides.CIFixModel)
	bind(flags, "ci-fix-reasoning-effort", &overrides.CIFixEffort)
	bind(flags, "ci-fix-prompt-file", &overrides.CIFixPromptPath)

	if len(args) > 0 && args[0] == "config" {
		args = append(append([]string{}, args[1:]...), "config")
	}
	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(stderr, "code-converge: %v\n", err)
		if !configCommand {
			logger := event.Logger{Out: stdout, Err: stderr, Now: a.Now, Format: "kv"}
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

	startupFormat := "kv"
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
	agent := codex.Adapter{Runner: processRunner, Config: cfg}
	logger := event.Logger{
		Out: stdout, Err: stderr, Now: a.Now, Format: cfg.LogFormat, Heartbeat: cfg.Heartbeat,
		Interactive: a.isTerminal(stdout), ColorDepth: a.colorDepth(cfg, stdout),
	}
	w := workflow.Workflow{Config: cfg, Agent: agent, Log: &logger, Err: stderr, Now: a.Now}
	return w.Run(ctx)
}

func (a App) isTerminal(out io.Writer) bool {
	if a.IsTerminal != nil {
		return a.IsTerminal(out)
	}
	file, ok := out.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
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
