package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/runner"
	selfupdate "github.com/dapi/code-converge/internal/update"
	"github.com/dapi/code-converge/internal/workflow"
)

func TestMain(m *testing.M) {
	clearGitRepositoryEnvironment()
	os.Exit(m.Run())
}

func clearGitRepositoryEnvironment() {
	for _, name := range []string{
		"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE",
		"GIT_OBJECT_DIRECTORY", "GIT_ALTERNATE_OBJECT_DIRECTORIES",
		"GIT_NAMESPACE", "GIT_CEILING_DIRECTORIES",
		"GIT_DISCOVERY_ACROSS_FILESYSTEM", "GIT_IMPLICIT_WORK_TREE",
	} {
		_ = os.Unsetenv(name)
	}
}

const cleanReviewJSONForApp = `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no findings","overall_confidence_score":0.99}`

func testRepo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if output, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitkeep"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("git", "-C", root, "add", ".gitkeep").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v: %s", err, output)
	}
	return root, t.TempDir()
}

func TestConfigCommand(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"config", "--mode=best", "--max-cycles=4"})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	for _, want := range []string{
		"log-format: human (built-in default)",
		"heartbeat: 0 (built-in default)",
		"color: auto (built-in default)",
		"mode: best (cli; built-in: fast)",
		"max-cycles: 4 (cli; built-in: 10)",
		"review-model: gpt-5.6-sol (best profile; built-in: gpt-5.6-terra)",
		"finalize-reasoning-effort: medium (best profile)",
		"ci-fix-reasoning-effort: high (best profile; built-in: medium)",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("missing %q in config output:\n%s", want, stdout.String())
		}
	}
	if t.Failed() {
		t.Fatalf("config output:\n%s", stdout.String())
	}
}

func TestHumanInvalidConfigurationUsesResolvedRenderer(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"--log-format=human", "--max-cycles=-1"})
	if code != workflow.ExitOperational || !strings.Contains(stdout.String(), "Failed due to an operational error") || strings.Contains(stdout.String(), "event=") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestInvalidLogFormatUsesHumanFallback(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"--log-format=json"})
	if code != workflow.ExitOperational || !strings.Contains(stdout.String(), "Failed due to an operational error") || strings.Contains(stdout.String(), "event=") || !strings.Contains(stderr.String(), "log-format must be one of") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestInvalidModeDoesNotInvokeCodex(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{t: t}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), []string{"--mode=unknown"})
	if code != workflow.ExitOperational || len(fake.invocations) != 0 || !strings.Contains(stderr.String(), "mode must be one of") {
		t.Fatalf("code=%d invocations=%d stderr=%q", code, len(fake.invocations), stderr.String())
	}
}

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr}).Run(context.Background(), []string{"--version"})
	if code != 0 || stdout.String() != "code-converge vdev\n" || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRootHelpAliasesExitBeforeOperationalSetup(t *testing.T) {
	for _, args := range [][]string{{"-h"}, {"--help"}} {
		t.Run(args[0], func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			fake := &appFakeRunner{t: t}
			updater := &appFakeUpdater{code: workflow.ExitOperational}
			code := (App{
				Stdout:  &stdout,
				Stderr:  &stderr,
				Runner:  fake,
				Updater: updater,
			}).Run(context.Background(), args)
			if code != workflow.ExitSuccess || stdout.String() != "usage: code-converge [flags] [config]\n" || stderr.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if len(fake.invocations) != 0 || updater.called {
				t.Fatalf("invocations=%#v updater.called=%v", fake.invocations, updater.called)
			}
		})
	}
}

func TestUpdateCommandDispatchesWithoutStartingWorkflow(t *testing.T) {
	var stdout, stderr bytes.Buffer
	updater := &appFakeUpdater{code: 0}
	code := (App{Stdout: &stdout, Stderr: &stderr, Updater: updater}).Run(context.Background(), []string{"update", "--yes"})
	if code != 0 || !updater.called || !updater.yes || stderr.Len() != 0 {
		t.Fatalf("code=%d called=%v yes=%v stderr=%q", code, updater.called, updater.yes, stderr.String())
	}
	if code := (App{Stdout: &stdout, Stderr: &stderr, Updater: updater}).Run(context.Background(), []string{"update", "--bad"}); code != workflow.ExitOperational {
		t.Fatalf("code=%d", code)
	}
}

type appFakeUpdater struct {
	called, yes bool
	code        int
}

func (u *appFakeUpdater) Run(_ context.Context, yes bool) int {
	u.called, u.yes = true, yes
	return u.code
}

var _ selfupdate.Runner = (*appFakeUpdater)(nil)

func TestInvalidConfigurationEmitsTerminalRun(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"--max-cycles=-1"})
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout.String(), "Failed due to an operational error") || strings.Contains(stdout.String(), "event=") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "non-negative integer") {
		t.Fatalf("stderr=%q", stderr.String())
	}
}

func TestInvalidFlagEmitsTerminalRun(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"--unknown"})
	if code != 2 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(stdout.String(), "Failed due to an operational error") || strings.Contains(stdout.String(), "event=") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestUnknownCommandFailsWithoutWorkflow(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"unknown"})
	if code != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "usage") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

type appFakeRunner struct {
	t             *testing.T
	invocations   []runner.Invocation
	review        runner.Result
	reviewMsg     string
	skipReviewMsg bool
	status        runner.Result
	statusErr     error
	finalizeMsg   string
	err           error
}

func (f *appFakeRunner) Run(_ context.Context, invocation runner.Invocation) (runner.Result, error) {
	f.invocations = append(f.invocations, invocation)
	if invocation.Executable == "gh" {
		return runner.Result{Stdout: "[]"}, nil
	}
	if invocation.Executable == "git" {
		args := strings.Join(invocation.Args, " ")
		switch {
		case strings.HasPrefix(args, "status "):
			return f.status, f.statusErr
		case args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case args == "config --get branch.feature.pushRemote", args == "config --get remote.pushDefault":
			return runner.Result{}, errors.New("not configured")
		case args == "config --get branch.feature.remote":
			return runner.Result{Stdout: "origin"}, nil
		case args == "remote get-url --push --all origin":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil
		case args == "remote get-url --all origin":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil
		case args == "config --get branch.feature.gh-merge-base":
			return runner.Result{}, errors.New("not configured")
		case args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case args == "symbolic-ref --quiet refs/remotes/origin/HEAD":
			return runner.Result{Stdout: "refs/remotes/origin/main"}, nil
		case strings.HasPrefix(args, "rev-parse --verify "):
			return runner.Result{Stdout: "0123456789012345678901234567890123456789"}, nil
		case strings.HasSuffix(args, "rev-parse --git-path index"):
			return runner.Result{Stdout: ".git/index"}, nil
		case strings.HasPrefix(args, "merge-base "):
			return runner.Result{Stdout: "abcdefabcdefabcdefabcdefabcdefabcdefabcd"}, nil
		case strings.HasPrefix(args, "read-tree ") || strings.HasSuffix(args, "-c core.splitIndex=false ls-files -v -z") || strings.HasSuffix(args, "add --sparse -A"):
			return runner.Result{}, nil
		}
		return f.status, f.statusErr
	}
	isReview := strings.Contains(invocation.Stdin, "prepared private Git index")
	for i, arg := range invocation.Args {
		if arg == "--output-last-message" && i+1 < len(invocation.Args) && f.err == nil {
			message := f.finalizeMsg
			if isReview {
				if f.skipReviewMsg {
					continue
				}
				message = f.reviewMsg
			}
			if err := os.WriteFile(invocation.Args[i+1], []byte(message), 0o600); err != nil {
				f.t.Fatalf("write output message: %v", err)
			}
		}
	}
	if isReview {
		return f.review, f.err
	}
	return runner.Result{}, f.err
}

func TestNilStreamsAndCwdDoNotPanic(t *testing.T) {
	root, home := testRepo(t)
	fake := &appFakeRunner{
		t:           t,
		review:      runner.Result{Stdout: "No findings.\n"},
		reviewMsg:   `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no findings","overall_confidence_score":0.99}`,
		finalizeMsg: `{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`,
	}
	code := (App{Cwd: root, Home: home, Runner: fake}).Run(context.Background(), nil)
	if code != workflow.ExitSuccess {
		t.Fatalf("code=%d", code)
	}
}

func TestConfigCommandInvalidFlagDoesNotEmitRunEvents(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home}).Run(context.Background(), []string{"config", "--unknown-flag"})
	if code != workflow.ExitOperational || stdout.Len() != 0 || !strings.Contains(stderr.String(), "flag provided") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunnerPassedFromAppIsUsed(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{t: t, err: errors.New("runner reached")}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), []string{"--log-format=kv"})
	if code != workflow.ExitOperational {
		t.Fatalf("code=%d", code)
	}
	if len(fake.invocations) == 0 {
		t.Fatal("app did not use injected runner")
	}
}

func TestAppWorkflowSuccessWithFakeRunner(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{
		t:           t,
		review:      runner.Result{Stdout: "No findings.\n"},
		reviewMsg:   `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no findings","overall_confidence_score":0.99}`,
		status:      runner.Result{Stdout: " M changed.go\n"},
		finalizeMsg: `{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`,
	}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), []string{"--log-format=kv"})
	if code != workflow.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "event=run_completed status=success exit_code=0") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if len(fake.invocations) < 2 {
		t.Fatalf("expected review and finalize invocations, got %d", len(fake.invocations))
	}
	var review runner.Invocation
	for _, invocation := range fake.invocations {
		if strings.Contains(invocation.Stdin, "prepared private Git index") {
			review = invocation
			break
		}
	}
	wrapperPath := ""
	for _, value := range review.Env {
		if strings.HasPrefix(value, "PATH=") {
			wrapperPath = strings.TrimPrefix(value, "PATH=")
			break
		}
	}
	reviewArgs := strings.Join(review.Args, " ")
	if len(review.Args) == 0 ||
		!strings.Contains(reviewArgs, " exec --output-schema ") ||
		!strings.Contains(reviewArgs, "--output-last-message") ||
		!strings.Contains(reviewArgs, "shell_environment_policy.set.PATH="+strconv.Quote(wrapperPath)) ||
		!strings.Contains(reviewArgs, "shell_environment_policy.set.SHELL="+strconv.Quote("/bin/sh")) ||
		!strings.Contains(reviewArgs, "shell_environment_policy.set.ZDOTDIR=") ||
		!strings.Contains(reviewArgs, "shell_environment_policy.set.BASH_ENV="+strconv.Quote("")) ||
		!strings.Contains(reviewArgs, "shell_environment_policy.set.ENV="+strconv.Quote("")) ||
		!strings.Contains(reviewArgs, "allow_login_shell=false") ||
		strings.Contains(reviewArgs, "GIT_INDEX_FILE") ||
		!strings.Contains(review.Stdin, "0123456789012345678901234567890123456789") ||
		!strings.Contains(review.Stdin, "git diff --cached abcdefabcdefabcdefabcdefabcdefabcdefabcd") ||
		wrapperPath == "" {
		t.Fatalf("review was not bound to the resolved target: %#v", review)
	}
}

func TestAppNoChangeSkipsFinalize(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{
		t:         t,
		review:    runner.Result{Stdout: "terminal output is not the result", Stderr: "terminal diagnostics are not the result"},
		reviewMsg: `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no changes","overall_confidence_score":0.99}`,
	}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), []string{"--log-format=kv"})
	if code != workflow.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "event=review_completed") || !strings.Contains(stdout.String(), "status=clean") || !strings.Contains(stdout.String(), "findings_total=0") || !strings.Contains(stdout.String(), "event=run_completed status=success exit_code=0") || strings.Contains(stdout.String(), "stage=finalize") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	last := fake.invocations[len(fake.invocations)-1]
	if last.Executable != "git" || !reflect.DeepEqual(last.Args, []string{"status", "--porcelain", "--untracked-files=all"}) {
		t.Fatalf("invocations = %#v", fake.invocations)
	}
}

func TestAppInvalidReviewResponseIsOperationalFailure(t *testing.T) {
	for _, test := range []struct {
		name       string
		reviewMsg  string
		skipOutput bool
		wantError  string
	}{
		{name: "missing", skipOutput: true, wantError: "read review response"},
		{name: "malformed", reviewMsg: "{", wantError: "parse structured review response"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, home := testRepo(t)
			var stdout, stderr bytes.Buffer
			fake := &appFakeRunner{
				t:             t,
				review:        runner.Result{Stdout: cleanReviewJSONForApp, Stderr: cleanReviewJSONForApp},
				reviewMsg:     test.reviewMsg,
				skipReviewMsg: test.skipOutput,
			}
			code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), []string{"--log-format=kv"})
			if code != workflow.ExitOperational ||
				!strings.Contains(stdout.String(), "event=run_completed status=operational_failure exit_code=2") ||
				!strings.Contains(stderr.String(), test.wantError) ||
				strings.Contains(stdout.String(), cleanReviewJSONForApp) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
		})
	}
}

func TestAppHumanNonTTYWorkflow(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{
		t:           t,
		review:      runner.Result{Stdout: "No findings.\n"},
		reviewMsg:   `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no findings","overall_confidence_score":0.99}`,
		finalizeMsg: `{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`,
	}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), nil)
	if code != workflow.ExitSuccess || !strings.Contains(stdout.String(), "Done (") || strings.Contains(stdout.String(), "\x1b") || strings.Contains(stdout.String(), "event=") || strings.Contains(stdout.String(), "No findings") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestAppCreatesHumanSessionLogHandoff(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{
		t:         t,
		review:    runner.Result{Stdout: "raw review output", Stderr: "raw review diagnostic"},
		reviewMsg: `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no changes","overall_confidence_score":0.99}`,
	}
	now := func() time.Time { return time.Date(2026, 7, 23, 22, 14, 5, 0, time.Local) }
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake, Now: now}).Run(context.Background(), nil)
	if code != workflow.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "22:14:05 Session log: ") {
		t.Fatalf("missing initial session handoff:\n%s", stdout.String())
	}
	path := strings.TrimPrefix(lines[0], "22:14:05 Session log: ")
	if !strings.HasPrefix(path, filepath.Join(home, ".code-converge", "session-logs", "session-")) {
		t.Fatalf("path=%q", path)
	}
	data, err := os.ReadFile(filepath.Join(path, "0001-invocation.json"))
	if err != nil || !strings.Contains(string(data), "raw review output") || strings.Contains(stdout.String(), "raw review output") {
		t.Fatalf("record=%q err=%v stdout=%q", data, err, stdout.String())
	}
}

func TestAppNoSessionLogCreatesNoArtifactsOrHandoff(t *testing.T) {
	root, home := testRepo(t)
	var stdout, stderr bytes.Buffer
	fake := &appFakeRunner{
		t:         t,
		review:    runner.Result{Stdout: "raw review output"},
		reviewMsg: `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no changes","overall_confidence_score":0.99}`,
	}
	code := (App{Stdout: &stdout, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), []string{"--no-session-log", "--log-format=kv"})
	if code != workflow.ExitSuccess || stderr.Len() != 0 || strings.Contains(stdout.String(), "Session log:") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".code-converge", "session-logs")); !os.IsNotExist(err) {
		t.Fatalf("session logs created: %v", err)
	}
}

func TestColorDepth(t *testing.T) {
	lookup := func(values map[string]string) func(string) (string, bool) {
		return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
	}
	terminal := func(io.Writer) bool { return true }
	for _, test := range []struct {
		name  string
		cfg   config.Config
		env   map[string]string
		depth int
	}{
		{"truecolor", config.Config{Color: "auto"}, map[string]string{"TERM": "xterm-256color", "COLORTERM": "truecolor"}, 3},
		{"ansi256", config.Config{Color: "auto"}, map[string]string{"TERM": "xterm-256color"}, 2},
		{"basic", config.Config{Color: "auto"}, map[string]string{"TERM": "xterm"}, 1},
		{"no color", config.Config{Color: "auto"}, map[string]string{"TERM": "xterm", "NO_COLOR": ""}, 0},
		{"never", config.Config{Color: "never"}, map[string]string{"TERM": "xterm"}, 0},
		{"dumb", config.Config{Color: "auto"}, map[string]string{"TERM": "dumb"}, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := App{IsTerminal: terminal, LookupEnv: lookup(test.env)}
			if got := app.colorDepth(test.cfg, &bytes.Buffer{}); got != test.depth {
				t.Fatalf("depth=%d, want %d", got, test.depth)
			}
		})
	}
}

func TestTerminalWidthUsesInjection(t *testing.T) {
	var got io.Writer
	want := &bytes.Buffer{}
	app := App{TerminalWidth: func(out io.Writer) (int, error) {
		got = out
		return 123, nil
	}}
	width, err := app.terminalWidth(want)
	if err != nil || width != 123 || got != want {
		t.Fatalf("width=%d err=%v writer=%p, want 123 nil %p", width, err, got, want)
	}
}

func TestIsTerminalRejectsNonTTYCharacterDevice(t *testing.T) {
	device, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	if (App{}).isTerminal(device) {
		t.Fatal("/dev/null was treated as an interactive terminal")
	}
}

func TestAppHumanDevNullWorkflow(t *testing.T) {
	root, home := testRepo(t)
	device, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	var stderr bytes.Buffer
	fake := &appFakeRunner{
		t:           t,
		review:      runner.Result{Stdout: "No findings.\n"},
		reviewMsg:   `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"no findings","overall_confidence_score":0.99}`,
		finalizeMsg: `{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`,
	}
	code := (App{Stdout: device, Stderr: &stderr, Cwd: root, Home: home, Runner: fake}).Run(context.Background(), nil)
	if code != workflow.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
}
