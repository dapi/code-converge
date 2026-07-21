package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/dapi/reviewer/internal/codex"
	"github.com/dapi/reviewer/internal/config"
	"github.com/dapi/reviewer/internal/event"
	"github.com/dapi/reviewer/internal/runner"
	"github.com/dapi/reviewer/internal/workflow"
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
	Stdout io.Writer
	Stderr io.Writer
	Cwd    string
	Home   string
	Runner runner.Runner
	Now    func() time.Time
}

func (a App) Run(ctx context.Context, args []string) int {
	stdout, stderr := a.Stdout, a.Stderr
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	cwd := a.Cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "reviewer: current directory: %v\n", err)
			return workflow.ExitOperational
		}
	}

	overrides := config.Overrides{}
	flags := flag.NewFlagSet("reviewer", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	bind(flags, "max-cycles", &overrides.MaxCycles)
	bind(flags, "max-ci-recoveries", &overrides.MaxCIRecoveries)
	bind(flags, "review-model", &overrides.ReviewModel)
	bind(flags, "review-reasoning-effort", &overrides.ReviewEffort)
	bind(flags, "fix-model", &overrides.FixModel)
	bind(flags, "fix-reasoning-effort", &overrides.FixEffort)
	bind(flags, "fix-prompt-file", &overrides.FixPromptPath)
	bind(flags, "finalize-model", &overrides.FinalizeModel)
	bind(flags, "finalize-prompt-file", &overrides.FinalizePromptPath)
	bind(flags, "ci-fix-model", &overrides.CIFixModel)
	bind(flags, "ci-fix-prompt-file", &overrides.CIFixPromptPath)

	if len(args) > 0 && args[0] == "config" {
		args = append(append([]string{}, args[1:]...), "config")
	}
	if err := flags.Parse(args); err != nil {
		fmt.Fprintf(stderr, "reviewer: %v\n", err)
		return workflow.ExitOperational
	}
	if flags.NArg() > 1 || (flags.NArg() == 1 && flags.Arg(0) != "config") {
		fmt.Fprintln(stderr, "reviewer: usage: reviewer [flags] [config]")
		return workflow.ExitOperational
	}

	cfg, err := config.Load(cwd, a.Home, overrides)
	if err != nil {
		if flags.NArg() == 1 {
			fmt.Fprintf(stderr, "reviewer: configuration: %v\n", err)
			return workflow.ExitOperational
		}
		logger := event.Logger{Out: stdout, Now: a.Now}
		started := time.Now()
		_ = logger.Emit("run_started")
		fmt.Fprintf(stderr, "reviewer: configuration: %v\n", err)
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
	return workflow.Workflow{Config: cfg, Agent: agent, Log: event.Logger{Out: stdout, Now: a.Now}, Err: stderr, Now: a.Now}.Run(ctx)
}

func bind(flags *flag.FlagSet, name string, target *config.OptionalString) {
	flags.Var(optionalFlag{target: target}, name, strings.ReplaceAll(name, "-", " "))
}
