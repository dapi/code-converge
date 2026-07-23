package repository

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dapi/code-converge/internal/runner"
)

type scriptedRunner struct {
	t           *testing.T
	invocations []runner.Invocation
	run         func(runner.Invocation) (runner.Result, error)
}

func (s *scriptedRunner) Run(_ context.Context, invocation runner.Invocation) (runner.Result, error) {
	s.invocations = append(s.invocations, invocation)
	return s.run(invocation)
}

type fakeRunner struct {
	result      runner.Result
	err         error
	invocations []runner.Invocation
}

func (f *fakeRunner) Run(_ context.Context, invocation runner.Invocation) (runner.Result, error) {
	f.invocations = append(f.invocations, invocation)
	return f.result, f.err
}

func TestStatusHasChanges(t *testing.T) {
	for _, test := range []struct {
		name string
		out  string
		want bool
	}{
		{"clean", "", false},
		{"whitespace clean", " \n", false},
		{"staged", "M  file.go\n", true},
		{"unstaged", " M file.go\n", true},
		{"untracked", "?? file.go\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeRunner{result: runner.Result{Stdout: test.out}}
			got, err := (Status{Runner: fake}).HasChanges(context.Background())
			if err != nil || got != test.want {
				t.Fatalf("HasChanges() = %v, %v; want %v, nil", got, err, test.want)
			}
			wantInvocation := runner.Invocation{Executable: "git", Args: []string{"status", "--porcelain"}}
			if !reflect.DeepEqual(fake.invocations, []runner.Invocation{wantInvocation}) {
				t.Fatalf("invocations = %#v", fake.invocations)
			}
		})
	}
}

func TestStatusPropagatesRunnerError(t *testing.T) {
	fake := &fakeRunner{err: errors.New("git unavailable")}
	if _, err := (Status{Runner: fake}).HasChanges(context.Background()); err == nil {
		t.Fatal("expected runner error")
	}
}

func TestStatusCheckpointCommitsLocallyWithoutPush(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "status --porcelain":
			return runner.Result{Stdout: " M fixed.go\n"}, nil
		case "add -A", "commit -m chore: checkpoint review fixes":
			return runner.Result{}, nil
		case "branch --show-current":
			return runner.Result{Stdout: "feature/checkpoints\n"}, nil
		case "rev-parse --short HEAD":
			return runner.Result{Stdout: "abc1234\n"}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	checkpoint, err := (Status{Runner: fake}).Checkpoint(context.Background())
	if err != nil || checkpoint != (Checkpoint{Created: true, Branch: "feature/checkpoints", Commit: "abc1234"}) {
		t.Fatalf("checkpoint=%#v err=%v", checkpoint, err)
	}
	for _, invocation := range fake.invocations {
		if len(invocation.Args) > 0 && invocation.Args[0] == "push" {
			t.Fatalf("checkpoint pushed unexpectedly: %#v", fake.invocations)
		}
	}
}

func TestStatusCheckpointSkipsEmptyCommit(t *testing.T) {
	fake := &fakeRunner{result: runner.Result{Stdout: ""}}
	checkpoint, err := (Status{Runner: fake}).Checkpoint(context.Background())
	if err != nil || checkpoint.Created || len(fake.invocations) != 1 {
		t.Fatalf("checkpoint=%#v err=%v invocations=%#v", checkpoint, err, fake.invocations)
	}
}

func TestStatusCheckpointPropagatesCommitFailure(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "status --porcelain":
			return runner.Result{Stdout: " M fixed.go\n"}, nil
		case "add -A":
			return runner.Result{}, nil
		case "commit -m chore: checkpoint review fixes":
			return runner.Result{}, errors.New("author identity unknown")
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (Status{Runner: fake}).Checkpoint(context.Background())
	if err == nil || !strings.Contains(err.Error(), "commit findings checkpoint") {
		t.Fatalf("error=%v", err)
	}
}

func TestStatusIsCleanRecognizesExistingWork(t *testing.T) {
	fake := &fakeRunner{result: runner.Result{Stdout: " M user-work.go\n"}}
	clean, err := (Status{Runner: fake}).IsClean(context.Background())
	if err != nil || clean {
		t.Fatalf("clean=%v error=%v", clean, err)
	}
}

func TestReviewScopeExplicitBaseBuildsPrivateSnapshot(t *testing.T) {
	fake := &scriptedRunner{t: t}
	fake.run = func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "rev-parse --verify develop^{commit}", "merge-base HEAD develop", "read-tree deadbeef", "add -A":
			if len(inv.Env) > 0 && strings.HasPrefix(strings.Join(inv.Args, " "), "read-tree") || strings.Join(inv.Args, " ") == "add -A" {
				if len(inv.Env) != 1 || !strings.HasPrefix(inv.Env[0], "GIT_INDEX_FILE=") {
					t.Fatalf("snapshot env=%#v", inv.Env)
				}
			}
			if strings.HasPrefix(strings.Join(inv.Args, " "), "rev-parse") {
				return runner.Result{Stdout: "deadbeef"}, nil
			}
			if strings.HasPrefix(strings.Join(inv.Args, " "), "merge-base") {
				return runner.Result{Stdout: "deadbeef"}, nil
			}
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}
	scope := &ReviewScope{Runner: fake, Base: "develop", copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}
	target, err := scope.Prepare(context.Background())
	if err != nil || target.Source != "explicit" || target.Base != "develop" || target.MergeBase != "deadbeef" || len(target.Env) != 1 {
		t.Fatalf("target=%#v err=%v", target, err)
	}
	if err := scope.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReviewScopeRejectsMultipleOpenPRs(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch inv.Executable {
		case "git":
			return runner.Result{Stdout: "feature"}, nil
		case "gh":
			return runner.Result{Stdout: `[{"baseRefName":"main"},{"baseRefName":"develop"}]`}, nil
		default:
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).Prepare(context.Background())
	if err == nil || !strings.Contains(err.Error(), "2 open pull requests") {
		t.Fatalf("error=%v", err)
	}
}

func TestReviewScopeUsesUniqueRemoteTrackingPRBase(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"develop","baseRefOid":"base-sha"}]`}, nil
		case args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case args == "rev-parse --verify develop^{commit}":
			return runner.Result{}, errors.New("no local branch")
		case args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case args == "rev-parse --verify refs/remotes/origin/develop^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case args == "merge-base HEAD refs/remotes/origin/develop":
			return runner.Result{Stdout: "merge-sha"}, nil
		case args == "read-tree merge-sha" || args == "add -A":
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}).Prepare(context.Background())
	if err != nil || target.Source != "open_pr" || target.Base != "refs/remotes/origin/develop" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}

func TestReviewScopeUsesSlashContainingPRBase(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"release/1.x","baseRefOid":"base-sha"}]`}, nil
		case args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case args == "rev-parse --verify refs/remotes/origin/release/1.x^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case args == "merge-base HEAD refs/remotes/origin/release/1.x":
			return runner.Result{Stdout: "merge-sha"}, nil
		case args == "add -A":
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}).Prepare(context.Background())
	if err != nil || target.Base != "refs/remotes/origin/release/1.x" || target.BaseCommit != "base-sha" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}

func TestReviewScopeRejectsStaleProviderBase(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"provider-sha"}]`}, nil
		case args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case args == "rev-parse --verify refs/remotes/origin/main^{commit}":
			return runner.Result{Stdout: "stale-sha"}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).Prepare(context.Background())
	if err == nil || !strings.Contains(err.Error(), "is stale") || !strings.Contains(err.Error(), "fetch") {
		t.Fatalf("error=%v", err)
	}
}

func TestReviewScopePreservesCommittedChangesOutsideSparseCheckout(t *testing.T) {
	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	runGit("init", "-q")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	for path, content := range map[string]string{"included/a.txt": "base", "excluded/b.txt": "base"} {
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	runGit("add", "-A")
	runGit("commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(root, "excluded", "b.txt"), []byte("branch change"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "-A")
	runGit("commit", "-qm", "branch")
	runGit("sparse-checkout", "init", "--cone")
	runGit("sparse-checkout", "set", "included")

	scope := &ReviewScope{Runner: runner.Exec{Dir: root}, Root: root, Base: "HEAD~1"}
	target, err := scope.Prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	command := exec.Command("git", "-C", root, "diff", "--cached", "--name-status", "HEAD~1")
	command.Env = append(os.Environ(), target.Env...)
	output, err := command.Output()
	if err != nil || !strings.Contains(string(output), "M\texcluded/b.txt") {
		t.Fatalf("sparse snapshot diff=%q err=%v", output, err)
	}
}
