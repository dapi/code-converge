package repository

import (
	"context"
	"errors"
	"fmt"
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

func TestMain(m *testing.M) {
	if IsScopedGitWrapperInvocation(os.Args[0]) {
		os.Exit(RunScopedGitWrapper(os.Args[1:]))
	}
	clearGitRepositoryEnvironment()
	os.Exit(m.Run())
}

// Real-Git tests create disposable repositories. They must not inherit an
// alternate index or another repository selector from the caller.
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

func isSnapshotAdd(args string) bool {
	return args == "-c "+disableSplitIndexConfig+" ls-files -v -z" ||
		args == "-c "+disableSplitIndexConfig+" add --sparse -A"
}

func currentProviderResult(args string) (runner.Result, error, bool) {
	switch args {
	case "config --get branch.feature.pushRemote", "config --get remote.pushDefault":
		return runner.Result{}, errors.New("not configured"), true
	case "config --get branch.feature.remote":
		return runner.Result{Stdout: "origin"}, nil, true
	case "remote get-url --push --all origin":
		return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil, true
	case "remote get-url --all origin":
		return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil, true
	case "remote":
		return runner.Result{Stdout: "origin"}, nil, true
	default:
		return runner.Result{}, nil, false
	}
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
			wantInvocation := runner.Invocation{Executable: "git", Args: []string{"status", "--porcelain", "--untracked-files=all"}}
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
		case "status --porcelain --untracked-files=all":
			return runner.Result{Stdout: " M fixed.go\n"}, nil
		case "add -A", "commit -m chore: checkpoint review fixes":
			return runner.Result{}, nil
		case "rev-parse HEAD":
			return runner.Result{Stdout: "new-sha\n"}, nil
		case "branch --show-current":
			return runner.Result{Stdout: "feature/checkpoints\n"}, nil
		case "rev-parse --short HEAD":
			return runner.Result{Stdout: "abc1234\n"}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	checkpoint, err := (Status{Runner: fake}).Checkpoint(context.Background(), "old-sha", true)
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
	checkpoint, err := (Status{Runner: fake}).Checkpoint(context.Background(), "", true)
	if err != nil || checkpoint.Created || len(fake.invocations) != 2 {
		t.Fatalf("checkpoint=%#v err=%v invocations=%#v", checkpoint, err, fake.invocations)
	}
}

func TestStatusCheckpointPropagatesCommitFailure(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "status --porcelain --untracked-files=all":
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
	_, err := (Status{Runner: fake}).Checkpoint(context.Background(), "old-sha", true)
	if err == nil || !strings.Contains(err.Error(), "commit findings checkpoint") {
		t.Fatalf("error=%v", err)
	}
}

func TestStatusCheckpointDetectsAgentCommitOnCleanWorktree(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "status --porcelain --untracked-files=all":
			return runner.Result{}, nil
		case "rev-parse HEAD":
			return runner.Result{Stdout: "new-sha\n"}, nil
		case "branch --show-current":
			return runner.Result{Stdout: "feature/fix\n"}, nil
		case "rev-parse --short HEAD":
			return runner.Result{Stdout: "abc1234\n"}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	checkpoint, err := (Status{Runner: fake}).Checkpoint(context.Background(), "old-sha", false)
	if err != nil || checkpoint != (Checkpoint{Created: true, Branch: "feature/fix", Commit: "abc1234"}) {
		t.Fatalf("checkpoint=%#v err=%v", checkpoint, err)
	}
	for _, invocation := range fake.invocations {
		if len(invocation.Args) > 0 && (invocation.Args[0] == "add" || invocation.Args[0] == "commit") {
			t.Fatalf("agent commit detection created an extra checkpoint: %#v", fake.invocations)
		}
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
		case "rev-parse --verify develop^{commit}", "merge-base HEAD deadbeef", "read-tree deadbeef", "-c " + disableSplitIndexConfig + " ls-files -v -z", "-c " + disableSplitIndexConfig + " add --sparse -A":
			if len(inv.Env) > 0 && strings.HasPrefix(strings.Join(inv.Args, " "), "read-tree") || isSnapshotAdd(strings.Join(inv.Args, " ")) {
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
	if err != nil || target.Source != "explicit" || target.Base != "develop" || target.MergeBase != "deadbeef" || len(target.Env) != 1 || !strings.HasPrefix(target.Env[0], "PATH=") {
		t.Fatalf("target=%#v err=%v", target, err)
	}
	if err := scope.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestReviewScopeFallsBackForGitWithoutSparseAddOutsideSparseCheckout(t *testing.T) {
	addSparseErr := errors.New("git add does not support --sparse")
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "-c " + disableSplitIndexConfig + " add --sparse -A":
			return runner.Result{Stderr: "error: unknown option `sparse'", ExitCode: 129}, addSparseErr
		case "config --bool core.sparseCheckout":
			return runner.Result{ExitCode: 1}, errors.New("git config key is unset")
		case "-c " + disableSplitIndexConfig + " add -A":
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	scope := &ReviewScope{Runner: fake, tempDir: t.TempDir()}
	if err := scope.snapshotWorktree(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestReviewScopeRequiresSparseAddForSparseCheckout(t *testing.T) {
	addSparseErr := errors.New("git add does not support --sparse")
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "-c " + disableSplitIndexConfig + " add --sparse -A":
			return runner.Result{Stderr: "error: unknown option `sparse'", ExitCode: 129}, addSparseErr
		case "config --bool core.sparseCheckout":
			return runner.Result{Stdout: "true", ExitCode: 0}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	scope := &ReviewScope{Runner: fake, tempDir: t.TempDir()}
	err := scope.snapshotWorktree(context.Background())
	if err == nil || !strings.Contains(err.Error(), "sparse checkout requires Git with git add --sparse support") {
		t.Fatalf("snapshotWorktree() error = %v", err)
	}
}

func TestReviewScopeMakesTemporaryDirectoryAbsolute(t *testing.T) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(workingDirectory); err != nil {
			t.Error(err)
		}
	})
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("relative-tmp", 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", "relative-tmp")
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "rev-parse --verify develop^{commit}", "merge-base HEAD deadbeef":
			return runner.Result{Stdout: "deadbeef"}, nil
		case "-c " + disableSplitIndexConfig + " ls-files -v -z", "-c " + disableSplitIndexConfig + " add --sparse -A":
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	scope := &ReviewScope{Runner: fake, Base: "develop", copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}
	target, err := scope.Prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	if !filepath.IsAbs(scope.tempDir) {
		t.Fatalf("temporary review directory = %q, want absolute path", scope.tempDir)
	}
	path, ok := reviewEnvironmentValue(target.Env, "PATH")
	if !ok || !filepath.IsAbs(strings.Split(path, string(os.PathListSeparator))[0]) {
		t.Fatalf("review PATH = %q, want an absolute wrapper directory", path)
	}
}

func TestReviewScopeRelocatesTemporaryDirectoryInsideReviewedWorktree(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".tmp"), 0o700); err != nil {
		t.Fatal(err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", ".tmp")

	scope := &ReviewScope{Root: root}
	dir, err := scope.createReviewTempDir()
	if err != nil {
		t.Fatal(err)
	}
	scope.tempDir = dir
	defer scope.Close()
	canonicalRoot, err := canonicalPath(root)
	if err != nil {
		t.Fatal(err)
	}
	if pathWithin(scope.tempDir, canonicalRoot) {
		t.Fatalf("temporary review directory %q is inside reviewed worktree %q", scope.tempDir, canonicalRoot)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".tmp"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "code-converge-review-index-") {
			t.Fatalf("unsafe temporary directory %q was not removed", entry.Name())
		}
	}
}

func TestReviewScopeStoresAbsoluteGitExecutable(t *testing.T) {
	git := mustGitExecutable(t)
	if !filepath.IsAbs(git) {
		var err error
		git, err = filepath.Abs(git)
		if err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	tools := filepath.Join(root, "tools")
	if err := os.Mkdir(tools, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(git, filepath.Join(tools, "git")); err != nil {
		t.Fatal(err)
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(workingDirectory) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", "tools"+string(os.PathListSeparator)+os.Getenv("PATH"))
	scope := &ReviewScope{tempDir: t.TempDir()}
	if _, err := scope.reviewEnvironment(); err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	configuration, ok := scopedGitConfigurationForInvocation(filepath.Join(scope.gitWrapperDir, "git"))
	if !ok {
		t.Fatal("scoped git wrapper configuration is unavailable")
	}
	if !filepath.IsAbs(configuration.Executable) {
		t.Fatalf("scoped git executable = %q, want absolute path", configuration.Executable)
	}
}

func TestReviewScopeRejectsMultipleOpenPRs(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		if result, err, ok := currentProviderResult(args); ok {
			return result, err
		}
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"main-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}},{"baseRefName":"develop","baseRefOid":"develop-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}}]`}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).Prepare(context.Background())
	if err == nil || !strings.Contains(err.Error(), "2 open pull requests") {
		t.Fatalf("error=%v", err)
	}
}

func TestReviewScopeRejectsProviderMatchWhenAnotherTargetIsUnavailable(t *testing.T) {
	const fork = "github.com/contributor/code-converge"
	const upstream = "github.com/dapi/code-converge"
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "fork"}, nil
		case inv.Executable == "git" && args == "remote get-url --push --all fork":
			return runner.Result{Stdout: "git@github.com:contributor/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "fork\nupstream"}, nil
		case inv.Executable == "git" && args == "remote get-url --all fork":
			return runner.Result{Stdout: "git@github.com:contributor/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote get-url --all upstream":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+fork+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository --limit "+ghPRListLimit:
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"contributor/code-converge"}}]`}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+upstream+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository --limit "+ghPRListLimit:
			return runner.Result{}, errors.New("gh authentication unavailable")
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, found, err := (&ReviewScope{Runner: fake}).openPRBase(context.Background(), "feature")
	if err == nil || found || !strings.Contains(err.Error(), "another provider target returned a match") || !strings.Contains(err.Error(), upstream) {
		t.Fatalf("found=%v err=%v", found, err)
	}
}

func TestReviewScopeUsesUniqueRemoteTrackingPRBase(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		if result, err, ok := currentProviderResult(args); ok {
			return result, err
		}
		switch {
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"develop","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}}]`}, nil
		case args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case args == "rev-parse --verify develop^{commit}":
			return runner.Result{}, errors.New("no local branch")
		case args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case args == "rev-parse --verify refs/remotes/origin/develop^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case args == "merge-base HEAD base-sha":
			return runner.Result{Stdout: "merge-sha"}, nil
		case args == "read-tree merge-sha" || isSnapshotAdd(args):
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
		if result, err, ok := currentProviderResult(args); ok {
			return result, err
		}
		switch {
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"release/1.x","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}}]`}, nil
		case args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case args == "rev-parse --verify refs/remotes/origin/release/1.x^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case args == "merge-base HEAD base-sha":
			return runner.Result{Stdout: "merge-sha"}, nil
		case isSnapshotAdd(args):
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

func TestReviewScopeUsesSlashContainingBranchMergeBase(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "origin"}, nil
		case inv.Executable == "git" && args == "remote get-url --push --all origin":
			return runner.Result{}, nil
		case inv.Executable == "git" && args == "config --get branch.feature.gh-merge-base":
			return runner.Result{Stdout: "release/1.0"}, nil
		case inv.Executable == "git" && args == "rev-parse --verify release/1.0^{commit}":
			return runner.Result{}, errors.New("no local branch")
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case inv.Executable == "git" && args == "rev-parse --verify refs/remotes/origin/release/1.0^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case inv.Executable == "git" && args == "merge-base HEAD base-sha":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && isSnapshotAdd(args):
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}).Prepare(context.Background())
	if err != nil || target.Source != "branch_merge_base" || target.Base != "refs/remotes/origin/release/1.0" || target.BaseCommit != "base-sha" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}

func TestReviewScopeRejectsStaleProviderBase(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		if result, err, ok := currentProviderResult(args); ok {
			return result, err
		}
		switch {
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"provider-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}}]`}, nil
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

func TestReviewScopeRejectsPRFromUnrelatedForkWithSameBranchName(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		if result, err, ok := currentProviderResult(args); ok {
			return result, err
		}
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"someone-else/code-converge"}}]`}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).Prepare(context.Background())
	if err == nil || !strings.Contains(err.Error(), "does not match current branch provider") || !strings.Contains(err.Error(), "dapi/code-converge") {
		t.Fatalf("error=%v", err)
	}
}

func TestReviewScopeFallsBackWhenCurrentProviderCannotBeEstablished(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "git" && strings.HasPrefix(args, "config --get "):
			return runner.Result{}, errors.New("not configured")
		case inv.Executable == "git" && args == "remote get-url --push --all origin":
			return runner.Result{Stderr: "error: No such remote 'origin'"}, errors.New("git exited unsuccessfully")
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case inv.Executable == "git" && args == "symbolic-ref --quiet refs/remotes/origin/HEAD":
			return runner.Result{}, errors.New("no default ref")
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).Prepare(context.Background())
	if err == nil || !strings.Contains(err.Error(), "found 0 remote default branches") {
		t.Fatalf("error=%v", err)
	}
}

func TestReviewScopeFallsBackWhenConfiguredProviderRemoteIsUnavailable(t *testing.T) {
	for _, test := range []struct {
		name   string
		result runner.Result
		err    error
	}{
		{
			name:   "removed remote",
			result: runner.Result{Stderr: "error: No such remote 'removed'"},
			err:    errors.New("git exited unsuccessfully"),
		},
		{
			name:   "remote has no URL",
			result: runner.Result{Stderr: "fatal: No such URL found for remote: removed"},
			err:    errors.New("git exited unsuccessfully"),
		},
		{
			name: "URL-less remote",
			// Git can return an empty URL set for a configured remote without
			// a usable provider URL. It is handled as provider-unavailable.
			result: runner.Result{},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
				args := strings.Join(inv.Args, " ")
				switch {
				case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
					return runner.Result{Stdout: "feature"}, nil
				case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
					return runner.Result{Stdout: "removed"}, nil
				case inv.Executable == "git" && args == "remote get-url --push --all removed":
					if locale, ok := reviewEnvironmentValue(inv.Env, "LC_ALL"); !ok || locale != "C" {
						t.Fatalf("remote URL lookup locale = %q, %v; want C", locale, ok)
					}
					return test.result, test.err
				case inv.Executable == "git" && args == "config --get branch.feature.gh-merge-base":
					return runner.Result{Stdout: "main"}, nil
				case inv.Executable == "git" && args == "rev-parse --verify main^{commit}":
					return runner.Result{Stdout: "base-sha"}, nil
				case inv.Executable == "git" && args == "merge-base HEAD base-sha":
					return runner.Result{Stdout: "merge-sha"}, nil
				case inv.Executable == "git" && isSnapshotAdd(args):
					return runner.Result{}, nil
				case inv.Executable == "gh":
					t.Fatal("provider discovery should not run for an unavailable configured remote")
					return runner.Result{}, nil
				default:
					t.Fatalf("unexpected invocation: %#v", inv)
					return runner.Result{}, nil
				}
			}}
			target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
				return os.WriteFile(target, []byte("index"), 0o600)
			}}).Prepare(context.Background())
			if err != nil || target.Source != "branch_merge_base" || target.Base != "main" {
				t.Fatalf("target=%#v err=%v", target, err)
			}
		})
	}
}

func TestCurrentBranchProviderPropagatesConfiguredRemoteFailure(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "origin"}, nil
		case "remote get-url --push --all origin":
			return runner.Result{Stderr: "fatal: unable to read config file"}, errors.New("git exited unsuccessfully")
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).currentBranchProvider(context.Background(), "feature")
	if err == nil || !strings.Contains(err.Error(), "read push URL for remote \"origin\"") {
		t.Fatalf("error=%v", err)
	}
}

func TestCurrentBranchProviderPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "origin"}, nil
		case "remote get-url --push --all origin":
			return runner.Result{Stderr: "error: No such remote 'origin'"}, errors.New("git exited unsuccessfully")
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).currentBranchProvider(ctx, "feature")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v, want context cancellation", err)
	}
}

func TestReviewScopeUsesOriginWhenBranchHasNoConfiguredPushRemote(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "gh":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}}]`}, nil
		case inv.Executable == "git" && strings.HasPrefix(args, "config --get "):
			return runner.Result{}, errors.New("not configured")
		case inv.Executable == "git" && args == "remote get-url --push --all origin":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote get-url --all origin":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "origin"}, nil
		case inv.Executable == "git" && args == "rev-parse --verify refs/remotes/origin/main^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case inv.Executable == "git" && args == "merge-base HEAD base-sha":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && isSnapshotAdd(args):
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}).Prepare(context.Background())
	if err != nil || target.Source != "open_pr" || target.Base != "refs/remotes/origin/main" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}

func TestRepositoryIdentity(t *testing.T) {
	for _, test := range []struct {
		remote string
		want   string
	}{
		{"git@github.com:dapi/code-converge.git", "github.com/dapi/code-converge"},
		{"github.com:dapi/code-converge.git", "github.com/dapi/code-converge"},
		{"github-work:dapi/code-converge.git", "github-work/dapi/code-converge"},
		{"alice@github.example.com:owner/repo.git", "github.example.com/owner/repo"},
		{"ssh://git@github.example.com/dapi/code-converge.git", "github.example.com/dapi/code-converge"},
		{"ssh://git@example.test:2222/owner/repo.git", "example.test:2222/owner/repo"},
		{"ssh://git@example.test:22/owner/repo.git", "example.test/owner/repo"},
		{"ssh://git@[2001:db8::1]:2222/owner/repo.git", "[2001:db8::1]:2222/owner/repo"},
		{"ssh://git@[2001:db8::1]:22/owner/repo.git", "[2001:db8::1]/owner/repo"},
		{"https://[2001:db8::1]:443/owner/repo.git", "[2001:db8::1]/owner/repo"},
		{"git@[2001:db8::1]:owner/repo.git", "[2001:db8::1]/owner/repo"},
		{"https://github.com/dapi/code-converge.git", "github.com/dapi/code-converge"},
		{"git@GitHub.COM:Dapi/Code-Converge.git", "github.com/dapi/code-converge"},
	} {
		if got, err := repositoryIdentity(test.remote); err != nil || got != test.want {
			t.Errorf("repositoryIdentity(%q) = %q, %v; want %q", test.remote, got, err, test.want)
		}
	}
	for _, remote := range []string{
		"file:///tmp/code-converge.git",
		"file://localhost/owner/repository.git",
		"ftp://github.com/owner/repository.git",
		"git://github.com/owner/repository.git",
		"git@github.com:code-converge.git",
	} {
		if _, err := repositoryIdentity(remote); err == nil {
			t.Errorf("repositoryIdentity(%q) succeeded", remote)
		}
	}
}

func TestCurrentBranchProviderDeduplicatesCaseVariants(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		switch strings.Join(inv.Args, " ") {
		case "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "origin"}, nil
		case "remote get-url --push --all origin":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git\ngit@GitHub.COM:Dapi/Code-Converge.git"}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	identity, err := (&ReviewScope{Runner: fake}).currentBranchProvider(context.Background(), "feature")
	if err != nil || identity != "github.com/dapi/code-converge" {
		t.Fatalf("currentBranchProvider() = %q, %v", identity, err)
	}
}

func TestReviewScopeQueriesProviderAgainstPushRepositoryHost(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "enterprise"}, nil
		case inv.Executable == "git" && args == "remote get-url --push --all enterprise":
			return runner.Result{Stdout: "ssh://git@github.example.com:2222/dapi/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote get-url --all enterprise":
			return runner.Result{Stdout: "ssh://git@github.example.com:2222/dapi/code-converge.git"}, nil
		case inv.Executable == "gh" && args == "pr list --repo github.example.com:2222/dapi/code-converge --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository --limit "+ghPRListLimit:
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}}]`}, nil
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "enterprise"}, nil
		case inv.Executable == "git" && args == "rev-parse --verify refs/remotes/enterprise/main^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case inv.Executable == "git" && args == "merge-base HEAD base-sha":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && isSnapshotAdd(args):
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}).Prepare(context.Background())
	if err != nil || target.Base != "refs/remotes/enterprise/main" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}

func TestReviewScopeFindsForkPullRequestFromUpstreamRepository(t *testing.T) {
	const fork = "github.com/contributor/code-converge"
	const upstream = "github.com/dapi/code-converge"
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "fork"}, nil
		case inv.Executable == "git" && args == "remote get-url --push --all fork":
			return runner.Result{Stdout: "git@github.com:contributor/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "fork\nupstream"}, nil
		case inv.Executable == "git" && args == "remote get-url --all fork":
			return runner.Result{Stdout: "git@github.com:contributor/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote get-url --all upstream":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+fork+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository --limit "+ghPRListLimit:
			return runner.Result{Stdout: "[]"}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+upstream+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository --limit "+ghPRListLimit:
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"contributor/code-converge"}}]`}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	pr, found, err := (&ReviewScope{Runner: fake}).openPRBase(context.Background(), "feature")
	if err != nil || !found || pr.Name != "main" || pr.Commit != "base-sha" {
		t.Fatalf("pr=%#v found=%v err=%v", pr, found, err)
	}
}

func TestReviewScopeUsesUpstreamTrackingBaseForForkPullRequest(t *testing.T) {
	const fork = "github.com/contributor/code-converge"
	const upstream = "github.com/dapi/code-converge"
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "fork"}, nil
		case inv.Executable == "git" && args == "remote get-url --push --all fork":
			return runner.Result{Stdout: "git@github.com:contributor/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "fork\nupstream"}, nil
		case inv.Executable == "git" && args == "remote get-url --all fork":
			return runner.Result{Stdout: "git@github.com:contributor/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote get-url --all upstream":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git"}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+fork+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository --limit "+ghPRListLimit:
			return runner.Result{Stdout: "[]"}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+upstream+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository --limit "+ghPRListLimit:
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"contributor/code-converge"}}]`}, nil
		case inv.Executable == "git" && args == "rev-parse --verify refs/remotes/upstream/main^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case inv.Executable == "git" && args == "merge-base HEAD base-sha":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && isSnapshotAdd(args):
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}).Prepare(context.Background())
	if err != nil || target.Source != "open_pr" || target.Base != "refs/remotes/upstream/main" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}

func TestReviewScopeFallsBackFromNonProviderPushRemote(t *testing.T) {
	for _, remoteURL := range []string{"file:///tmp/code-converge.git", "https://gitlab.example.com/group/subgroup/code-converge.git"} {
		t.Run(remoteURL, func(t *testing.T) {
			fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
				args := strings.Join(inv.Args, " ")
				switch {
				case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
					return runner.Result{Stdout: "feature"}, nil
				case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
					return runner.Result{Stdout: "origin"}, nil
				case inv.Executable == "git" && args == "remote get-url --push --all origin":
					return runner.Result{Stdout: remoteURL}, nil
				case inv.Executable == "git" && args == "config --get branch.feature.gh-merge-base":
					return runner.Result{Stdout: "main"}, nil
				case inv.Executable == "git" && args == "rev-parse --verify main^{commit}":
					return runner.Result{Stdout: "base-sha"}, nil
				case inv.Executable == "git" && args == "merge-base HEAD base-sha":
					return runner.Result{Stdout: "merge-sha"}, nil
				case inv.Executable == "git" && isSnapshotAdd(args):
					return runner.Result{}, nil
				case inv.Executable == "gh":
					t.Fatal("provider discovery should not run for a non-provider push remote")
					return runner.Result{}, nil
				default:
					t.Fatalf("unexpected invocation: %#v", inv)
					return runner.Result{}, nil
				}
			}}
			target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
				return os.WriteFile(target, []byte("index"), 0o600)
			}}).Prepare(context.Background())
			if err != nil || target.Source != "branch_merge_base" || target.Base != "main" {
				t.Fatalf("target=%#v err=%v", target, err)
			}
		})
	}
}

func TestReviewScopeFallsBackFromMixedProviderAndNonProviderPushURLs(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "origin"}, nil
		case inv.Executable == "git" && args == "remote get-url --push --all origin":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git\nfile:///tmp/mirror.git"}, nil
		case inv.Executable == "git" && args == "config --get branch.feature.gh-merge-base":
			return runner.Result{Stdout: "main"}, nil
		case inv.Executable == "git" && args == "rev-parse --verify main^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case inv.Executable == "git" && args == "merge-base HEAD base-sha":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && isSnapshotAdd(args):
			return runner.Result{}, nil
		case inv.Executable == "gh":
			t.Fatal("provider discovery should not run for mixed push URLs")
			return runner.Result{}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	target, err := (&ReviewScope{Runner: fake, copyIndex: func(_ context.Context, target string) error {
		return os.WriteFile(target, []byte("index"), 0o600)
	}}).Prepare(context.Background())
	if err != nil || target.Source != "branch_merge_base" || target.Base != "main" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}

func TestReviewScopeRejectsConflictingPushRepositoryIdentities(t *testing.T) {
	fake := &scriptedRunner{t: t, run: func(inv runner.Invocation) (runner.Result, error) {
		args := strings.Join(inv.Args, " ")
		switch {
		case inv.Executable == "git" && args == "symbolic-ref --quiet --short HEAD":
			return runner.Result{Stdout: "feature"}, nil
		case inv.Executable == "git" && args == "config --get branch.feature.pushRemote":
			return runner.Result{Stdout: "origin"}, nil
		case inv.Executable == "git" && args == "remote get-url --push --all origin":
			return runner.Result{Stdout: "git@github.com:dapi/code-converge.git\ngit@github.com:other/code-converge.git"}, nil
		default:
			t.Fatalf("unexpected invocation: %#v", inv)
			return runner.Result{}, nil
		}
	}}
	_, err := (&ReviewScope{Runner: fake}).Prepare(context.Background())
	if err == nil || !strings.Contains(err.Error(), "conflicting push repository identities") {
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

func TestReviewScopeBuildsSnapshotFromRepositorySubdirectory(t *testing.T) {
	root := t.TempDir()
	subdirectory := filepath.Join(root, "nested")
	if err := os.Mkdir(subdirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit := func(args ...string) {
		t.Helper()
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	runGit("init", "-q")
	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "tracked.txt")
	runGit("commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(root, "tracked.txt"), []byte("worktree change"), 0o600); err != nil {
		t.Fatal(err)
	}

	scope := &ReviewScope{Runner: runner.Exec{Dir: subdirectory}, Root: root, Base: "HEAD"}
	if _, err := scope.Prepare(context.Background()); err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	result, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "git",
		Args:       []string{"diff", "--cached", "--name-status", "HEAD"},
		Env:        scope.snapshotEnvironment(),
	})
	if err != nil || !strings.Contains(result.Stdout, "M\ttracked.txt") {
		t.Fatalf("subdirectory snapshot diff=%q err=%v", result.Stdout, err)
	}
}

func TestReviewScopeStagesHiddenTrackedWorktreeChanges(t *testing.T) {
	for _, test := range []struct {
		name  string
		path  string
		setup func(t *testing.T, root string, runGit func(...string))
	}{
		{
			name: "sparse worktree entry",
			path: "excluded/b.txt",
			setup: func(t *testing.T, root string, runGit func(...string)) {
				t.Helper()
				runGit("sparse-checkout", "init", "--cone")
				runGit("sparse-checkout", "set", "included")
				if err := os.MkdirAll(filepath.Join(root, "excluded"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "assume unchanged entry",
			path: "hidden.txt",
			setup: func(t *testing.T, _ string, runGit func(...string)) {
				t.Helper()
				runGit("update-index", "--assume-unchanged", "hidden.txt")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
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
			for path, content := range map[string]string{"included/a.txt": "base", "excluded/b.txt": "base", "hidden.txt": "base"} {
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
			test.setup(t, root, runGit)
			if err := os.WriteFile(filepath.Join(root, test.path), []byte("worktree change"), 0o600); err != nil {
				t.Fatal(err)
			}
			scope := &ReviewScope{Runner: runner.Exec{Dir: root}, Root: root, Base: "HEAD"}
			_, err := scope.Prepare(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer scope.Close()
			result, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
				Executable: "git",
				Args:       []string{"diff", "--cached", "--name-status", "HEAD"},
				Env:        scope.snapshotEnvironment(),
			})
			if err != nil || !strings.Contains(result.Stdout, "M\t"+test.path) {
				t.Fatalf("hidden snapshot diff=%q err=%v", result.Stdout, err)
			}
			if err := exec.Command("git", "-C", root, "diff", "--cached", "--quiet").Run(); err != nil {
				t.Fatalf("real index was changed while snapshotting %q: %v", test.path, err)
			}
		})
	}
}

func TestReviewScopePrivateIndexSurvivesPathOnlyEnvironment(t *testing.T) {
	root := t.TempDir()
	runGit := func(dir string, args ...string) {
		t.Helper()
		if output, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git -C %s %v: %v: %s", dir, args, err, output)
		}
	}
	runGit(root, "init", "-q")
	runGit(root, "config", "user.email", "test@example.com")
	runGit(root, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(root, "add", "base.txt")
	runGit(root, "commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(root, "change.txt"), []byte("change"), 0o600); err != nil {
		t.Fatal(err)
	}

	scope := &ReviewScope{Runner: runner.Exec{Dir: root}, Root: root, Base: "HEAD"}
	target, err := scope.Prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	if len(target.Env) != 1 || !strings.HasPrefix(target.Env[0], "PATH=") {
		t.Fatalf("review environment is not PATH-only: %#v", target.Env)
	}
	privateIndex := filepath.Join(scope.tempDir, "index")
	wrapperPath, ok := reviewEnvironmentValue(target.Env, "PATH")
	if !ok {
		t.Fatalf("review environment has no wrapper PATH: %#v", target.Env)
	}
	wrapperInfo, err := os.Lstat(filepath.Join(strings.Split(wrapperPath, string(os.PathListSeparator))[0], "git"))
	if err != nil || wrapperInfo.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("review git helper is not an executable symlink: info=%v err=%v", wrapperInfo, err)
	}
	wrapperDir := strings.Split(wrapperPath, string(os.PathListSeparator))[0]
	configuration, ok := scopedGitConfigurationForInvocation(filepath.Join(wrapperDir, "git"))
	if !ok {
		t.Fatal("scoped git wrapper configuration is unavailable")
	}
	if configuration.HelperDir == wrapperDir {
		t.Fatal("scoped Git helpers share the PATH wrapper directory")
	}
	if helper, err := os.Lstat(filepath.Join(configuration.HelperDir, "git-diff")); err != nil || helper.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("scoped git-diff helper is not linked in GIT_EXEC_PATH: info=%v err=%v", helper, err)
	}
	for _, helper := range []string{"git-diff", "git-add"} {
		command := exec.Command("sh", "-c", "command -v \"$1\"", "sh", helper)
		command.Env = target.Env
		output, err := command.Output()
		if err == nil && filepath.Clean(strings.TrimSpace(string(output))) == filepath.Join(wrapperDir, helper) {
			t.Fatalf("direct %s resolves through the PATH wrapper directory", helper)
		}
	}
	before, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	// This is the effective environment of a non-login Codex shell with an
	// include_only = ["PATH"] policy. No scoped helper variable is available to
	// the wrapper.
	for _, gitCommand := range []string{
		"git diff --cached --name-only HEAD",
		"git -P diff --cached --name-only HEAD",
		"git --no-lazy-fetch diff --cached --name-only HEAD",
		"git --no-advice diff --cached --name-only HEAD",
	} {
		command := exec.Command("sh", "-c", gitCommand)
		command.Dir = root
		command.Env = target.Env
		output, err := command.Output()
		if err != nil || !strings.Contains(string(output), "change.txt") {
			t.Fatalf("reviewed repository did not use private index for %q: output=%q err=%v", gitCommand, output, err)
		}
	}
	for _, gitCommand := range []string{
		"git --attr-source=HEAD check-attr --all -- change.txt",
		"git --attr-source HEAD check-attr --all -- change.txt",
		"git --list-cmds=builtins",
	} {
		command := exec.Command("sh", "-c", gitCommand)
		command.Dir = root
		command.Env = target.Env
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("documented Git global option failed for %q: output=%q err=%v", gitCommand, output, err)
		}
	}
	// Git ignores aliases that reuse a built-in command name. The wrapper must
	// make the same precedence decision before classifying an alias; otherwise
	// this harmless-looking ignored alias would withhold the review index.
	runGit(root, "config", "alias.diff", "!git clone")
	builtinDiff := exec.Command("sh", "-c", "git diff --cached --name-only HEAD")
	builtinDiff.Dir = root
	builtinDiff.Env = target.Env
	output, err := builtinDiff.Output()
	if err != nil || !strings.Contains(string(output), "change.txt") {
		t.Fatalf("built-in diff did not use private index with ignored alias: output=%q err=%v", output, err)
	}
	unsupported := exec.Command("sh", "-c", "git --code-converge-unknown-global diff --cached")
	unsupported.Dir = root
	unsupported.Env = target.Env
	if output, err := unsupported.CombinedOutput(); err == nil || !strings.Contains(string(output), "unsupported or malformed Git global option") {
		t.Fatalf("unsupported global option did not fail closed: output=%q err=%v", output, err)
	}

	nested := filepath.Join(root, "nested")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	runGit(nested, "init", "-q")
	if err := os.WriteFile(filepath.Join(nested, ".gitkeep"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{
		"git -c core.filemode=false -C \"$1\" add .gitkeep",
		"git -c core.filemode=false --git-dir \"$1/.git\" --work-tree \"$1\" add .gitkeep",
		"git --namespace review -C \"$1\" add .gitkeep",
		"\"$2\" -C \"$1\" add .gitkeep",
	} {
		if _, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
			Executable: "sh", Args: []string{"-c", command, "sh", nested, mustGitExecutable(t)}, Env: target.Env,
		}); err != nil {
			t.Fatalf("nested git add %q: %v", command, err)
		}
	}
	if err := os.WriteFile(filepath.Join(nested, "alias.txt"), []byte("alias"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(root, "config", "alias.add-nested", `!git -C "$1" add alias.txt`)
	if _, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git add-nested \"$1\"", "sh", nested}, Env: target.Env,
	}); err != nil {
		t.Fatalf("nested git add through shell alias: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nested, ".git", "index")); err != nil {
		t.Fatalf("nested repository index was not created: %v", err)
	}
	after, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatal("nested git operation changed the private review index")
	}
	if err := os.WriteFile(filepath.Join(nested, "absolute.txt"), []byte("absolute"), 0o600); err != nil {
		t.Fatal(err)
	}
	// This shell alias contains positional shell expansion, so the wrapper
	// cannot safely classify it and withholds the disposable review index.
	runGit(root, "config", "alias.add-nested-absolute", "!"+mustGitExecutable(t)+` -C "$1" add absolute.txt`)
	if _, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git add-nested-absolute \"$1\"", "sh", nested}, Env: target.Env,
	}); err != nil {
		t.Fatalf("nested git add through absolute shell alias: %v", err)
	}
	nestedIndex, err := exec.Command("git", "-C", nested, "diff", "--cached", "--name-only").Output()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(nestedIndex), "absolute.txt") {
		t.Fatalf("absolute descendant did not use nested repository index: %q", nestedIndex)
	}
	afterAbsolute, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterAbsolute, before) {
		t.Fatal("absolute nested git operation changed the private review index")
	}

	// A read-only absolute Git shell alias can still select another repository.
	// It must use that repository's ordinary index rather than the review index.
	runGit(root, "config", "alias.other-status", "!"+mustGitExecutable(t)+" -C nested status --porcelain")
	wantOtherStatus, err := exec.Command("git", "-C", nested, "status", "--porcelain").Output()
	if err != nil {
		t.Fatal(err)
	}
	otherStatus, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git other-status"}, Env: target.Env,
	})
	if err != nil {
		t.Fatalf("other repository status through absolute shell alias: %v", err)
	}
	if otherStatus.Stdout != string(wantOtherStatus) {
		t.Fatalf("other repository status = %q, want %q", otherStatus.Stdout, wantOtherStatus)
	}
	afterOtherStatus, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterOtherStatus, before) {
		t.Fatal("other repository status changed the private review index")
	}

	// An external git-* command is not a built-in or an alias. It may invoke an
	// absolute Git executable against another repository, so it must not inherit
	// the disposable review index either.
	externalHelper := filepath.Join(configuration.HelperDir, "git-external-other-status")
	externalScript := fmt.Sprintf("#!/bin/sh\nexec %q -C \"$1\" status --porcelain\n", mustGitExecutable(t))
	if err := os.WriteFile(externalHelper, []byte(externalScript), 0o700); err != nil {
		t.Fatal(err)
	}
	externalStatus, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git external-other-status \"$1\"", "sh", nested}, Env: target.Env,
	})
	if err != nil {
		t.Fatalf("other repository status through external Git command: %v", err)
	}
	if externalStatus.Stdout != string(wantOtherStatus) {
		t.Fatalf("external Git command status = %q, want %q", externalStatus.Stdout, wantOtherStatus)
	}
	afterExternalStatus, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterExternalStatus, before) {
		t.Fatal("external Git command changed the private review index")
	}

	source := t.TempDir()
	runGit(source, "init", "-q")
	runGit(source, "config", "user.email", "test@example.com")
	runGit(source, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(source, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(source, "add", "source.txt")
	runGit(source, "commit", "-qm", "source")
	// Repository-creating aliases must not inherit the disposable review index.
	// Otherwise Git initializes the destination with the reviewed repository's
	// index, leaving the new repository/worktree inconsistent.
	runGit(root, "config", "alias.clone-repository", "clone")
	clone := filepath.Join(root, "clone")
	if _, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git clone-repository \"$1\" \"$2\"", "sh", source, clone}, Env: target.Env,
	}); err != nil {
		t.Fatalf("clone repository through alias: %v", err)
	}
	if _, err := os.Stat(filepath.Join(clone, ".git", "index")); err != nil {
		t.Fatalf("cloned repository index was not created: %v", err)
	}
	if status, err := exec.Command("git", "-C", clone, "status", "--porcelain").Output(); err != nil || len(status) != 0 {
		t.Fatalf("cloned repository status = %q, %v; want clean", status, err)
	}
	afterClone, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterClone, before) {
		t.Fatal("clone changed the private review index")
	}

	// Git interprets a backslash in a regular alias as escaping the next
	// character. The wrapper must recognize the resulting clone command too.
	runGit(root, "config", "alias.clone-escaped", `cl\one`)
	escapedClone := filepath.Join(root, "escaped-clone")
	if _, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git clone-escaped \"$1\" \"$2\"", "sh", source, escapedClone}, Env: target.Env,
	}); err != nil {
		t.Fatalf("clone repository through escaped alias: %v", err)
	}
	if _, err := os.Stat(filepath.Join(escapedClone, ".git", "index")); err != nil {
		t.Fatalf("escaped-alias cloned repository index was not created: %v", err)
	}
	if status, err := exec.Command("git", "-C", escapedClone, "status", "--porcelain").Output(); err != nil || len(status) != 0 {
		t.Fatalf("escaped-alias cloned repository status = %q, %v; want clean", status, err)
	}
	afterEscapedClone, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterEscapedClone, before) {
		t.Fatal("escaped-alias clone changed the private review index")
	}

	// An absolute Git executable bypasses the wrapper, but the outer shell
	// alias must still be recognized as repository creation so it does not
	// inherit the disposable command index.
	runGit(root, "config", "alias.clone-repository-absolute", "!"+mustGitExecutable(t)+" clone")
	absoluteClone := filepath.Join(root, "absolute-clone")
	if _, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git clone-repository-absolute \"$1\" \"$2\"", "sh", source, absoluteClone}, Env: target.Env,
	}); err != nil {
		t.Fatalf("clone repository through absolute shell alias: %v", err)
	}
	if _, err := os.Stat(filepath.Join(absoluteClone, ".git", "index")); err != nil {
		t.Fatalf("absolute-alias cloned repository index was not created: %v", err)
	}
	if status, err := exec.Command("git", "-C", absoluteClone, "status", "--porcelain").Output(); err != nil || len(status) != 0 {
		t.Fatalf("absolute-alias cloned repository status = %q, %v; want clean", status, err)
	}
	afterAbsoluteClone, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterAbsoluteClone, before) {
		t.Fatal("absolute-alias clone changed the private review index")
	}

	runGit(root, "config", "alias.add-worktree", "worktree add")
	worktree := filepath.Join(t.TempDir(), "worktree")
	if _, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh", Args: []string{"-c", "git add-worktree \"$1\" HEAD", "sh", worktree}, Env: target.Env,
	}); err != nil {
		t.Fatalf("create worktree through alias: %v", err)
	}
	if _, err := os.Stat(filepath.Join(worktree, ".git")); err != nil {
		t.Fatalf("worktree was not created: %v", err)
	}
	if status, err := exec.Command("git", "-C", worktree, "status", "--porcelain").Output(); err != nil || len(status) != 0 {
		t.Fatalf("worktree status = %q, %v; want clean", status, err)
	}
	afterWorktree, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterWorktree, before) {
		t.Fatal("worktree creation changed the private review index")
	}
}

func TestReviewScopeRemovesInheritedGitTransportEnvironment(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "base.txt")
	runGit("commit", "-qm", "base")
	if err := os.WriteFile(filepath.Join(root, "change.txt"), []byte("change"), 0o600); err != nil {
		t.Fatal(err)
	}

	t.Setenv("GIT_INDEX_FILE", filepath.Join(t.TempDir(), "foreign-index"))
	t.Setenv("GIT_EXEC_PATH", t.TempDir())
	t.Setenv("GIT_DIR", filepath.Join(t.TempDir(), "foreign-git-dir"))

	scope := &ReviewScope{Runner: runner.Exec{Dir: root}, Root: root, Base: "HEAD"}
	target, err := scope.Prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	if !reflect.DeepEqual(target.UnsetEnv, gitTransportEnvironment) {
		t.Fatalf("review environment removals = %#v, want %#v", target.UnsetEnv, gitTransportEnvironment)
	}
	result, err := (runner.Exec{Dir: root}).Run(context.Background(), runner.Invocation{
		Executable: "sh",
		Args: []string{"-c",
			"test -z \"$GIT_INDEX_FILE\" && test -z \"$GIT_EXEC_PATH\" && test -z \"$GIT_DIR\" && git diff --cached --name-only HEAD",
		},
		Env:      target.Env,
		UnsetEnv: target.UnsetEnv,
	})
	if err != nil || !strings.Contains(result.Stdout, "change.txt") {
		t.Fatalf("sanitized scoped review = %#v, %v; want change.txt", result, err)
	}
}

func TestReviewScopePrivateIndexDoesNotMaintainSplitIndex(t *testing.T) {
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
	if err := os.WriteFile(filepath.Join(root, "base.txt"), []byte("base"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "base.txt")
	runGit("commit", "-qm", "base")
	runGit("config", "core.splitIndex", "true")
	runGit("config", "splitIndex.sharedIndexExpire", "now")
	runGit("update-index", "--split-index")
	shared, err := filepath.Glob(filepath.Join(root, ".git", "sharedindex.*"))
	if err != nil || len(shared) == 0 {
		t.Fatalf("split index shared files = %q, %v; want at least one", shared, err)
	}
	if err := os.WriteFile(filepath.Join(root, "change.txt"), []byte("change"), 0o600); err != nil {
		t.Fatal(err)
	}
	scope := &ReviewScope{Runner: runner.Exec{Dir: root}, Root: root, Base: "HEAD"}
	target, err := scope.Prepare(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer scope.Close()
	command := exec.Command("sh", "-c", "git diff --cached --name-only HEAD")
	command.Dir = root
	command.Env = target.Env
	if output, err := command.CombinedOutput(); err != nil || !strings.Contains(string(output), "change.txt") {
		t.Fatalf("scoped private-index diff = %q, %v; want change.txt", output, err)
	}
	if output, err := exec.Command("git", "-C", root, "status", "--porcelain").CombinedOutput(); err != nil {
		t.Fatalf("real repository status failed after private snapshot: %v: %s", err, output)
	}
}

func mustGitExecutable(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	return git
}

func reviewEnvironmentValue(environment []string, name string) (string, bool) {
	prefix := name + "="
	for _, value := range environment {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix), true
		}
	}
	return "", false
}

func TestGitCreationAndTargetParsingConsumeNamespaceValue(t *testing.T) {
	prefix, subcommand, remainder, ok := splitGitGlobalOptions([]string{"--namespace", "review", "-C", "other", "add", "file"})
	if !ok || subcommand != "add" || !reflect.DeepEqual(prefix, []string{"--namespace", "review", "-C", "other"}) || !reflect.DeepEqual(remainder, []string{"file"}) {
		t.Fatalf("splitGitGlobalOptions() = %#v, %q, %#v, %v", prefix, subcommand, remainder, ok)
	}
	for _, args := range [][]string{
		{"--namespace", "review", "clone", "source", "destination"},
		{"--namespace", "review", "worktree", "add", "destination"},
	} {
		if !gitCreatesRepository(args, "", "") {
			t.Fatalf("gitCreatesRepository(%#v) = false", args)
		}
	}
}

func TestGitCreatesRepositoryResolvesAliases(t *testing.T) {
	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		if output, err := exec.Command("git", append([]string{"-C", root}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	runGit("init", "-q")
	git := mustGitExecutable(t)
	runGit("config", "alias.clone-repository", "clone")
	runGit("config", "alias.clone-escaped", `cl\one`)
	runGit("config", "alias.add-worktree", "worktree add")
	runGit("config", "alias.chained-worktree", "add-worktree")
	runGit("config", "alias.shell-worktree", `!f() { git worktree add "$@"; }; f`)
	runGit("config", "alias.absolute-shell-clone", "!"+git+" clone")
	runGit("config", "alias.other-status", "-C ../other status --porcelain")
	runGit("config", "alias.review-status", "status --short")

	for _, args := range [][]string{
		{"-C", root, "clone-repository", "source", "destination"},
		{"-C", root, "clone-escaped", "source", "destination"},
		{"-C", root, "add-worktree", "destination"},
		{"-C", root, "chained-worktree", "destination"},
		{"-C", root, "shell-worktree", "destination"},
		{"-C", root, "absolute-shell-clone", "source", "destination"},
	} {
		if !gitCreatesRepository(args, git, root) {
			t.Fatalf("gitCreatesRepository(%#v) = false; want true", args)
		}
	}
	if gitCreatesRepository([]string{"-C", root, "review-status"}, git, root) {
		t.Fatal("gitCreatesRepository() classified a non-creating alias as repository creation")
	}
	// This regular alias expands after the outer command has been classified.
	// Its target selection must withhold the reviewed repository's private index.
	if !gitCreatesRepository([]string{"other-status"}, git, root) {
		t.Fatal("gitCreatesRepository() did not withhold the index for a target-selecting regular alias")
	}
}

func TestLinkGitHelpersReservesScopedConfigurationFile(t *testing.T) {
	gitExecPath := t.TempDir()
	wrapperDir := t.TempDir()
	sourceConfiguration := filepath.Join(gitExecPath, scopedGitConfigurationFile)
	if err := os.WriteFile(sourceConfiguration, []byte("do not modify"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := linkGitHelpers(gitExecPath, wrapperDir); err != nil {
		t.Fatal(err)
	}
	destinationConfiguration := filepath.Join(wrapperDir, scopedGitConfigurationFile)
	if _, err := os.Lstat(destinationConfiguration); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reserved configuration file was linked: %v", err)
	}
	if err := os.WriteFile(destinationConfiguration, []byte("wrapper configuration"), 0o600); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(sourceConfiguration)
	if err != nil || string(content) != "do not modify" {
		t.Fatalf("git exec configuration = %q, %v; want original content", content, err)
	}
}

func TestReviewScopeRejectsTemporaryPathWithPathListSeparator(t *testing.T) {
	scope := &ReviewScope{tempDir: filepath.Join(t.TempDir(), "review"+string(os.PathListSeparator)+"index")}
	if _, err := scope.reviewEnvironment(); err == nil || !strings.Contains(err.Error(), "PATH list separator") {
		t.Fatalf("reviewEnvironment() error = %v, want path-list separator diagnostic", err)
	}
}

func TestSplitGitAliasArgumentsEscapes(t *testing.T) {
	for _, test := range []struct {
		input string
		want  []string
		ok    bool
	}{
		{input: `cl\one`, want: []string{"clone"}, ok: true},
		{input: `clone\ repository`, want: []string{"clone repository"}, ok: true},
		{input: `"cl\one"`, want: []string{"clone"}, ok: true},
		{input: `clone\`, ok: false},
	} {
		got, ok := splitGitAliasArguments(test.input)
		if ok != test.ok || !reflect.DeepEqual(got, test.want) {
			t.Fatalf("splitGitAliasArguments(%q) = %#v, %v; want %#v, %v", test.input, got, ok, test.want, test.ok)
		}
	}
}

func TestShellAliasCreatesRepositoryFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name      string
		expansion string
		want      bool
	}{
		{name: "absolute git clone", expansion: "!/usr/bin/git clone source destination", want: true},
		{name: "read only git command", expansion: "!git status --short", want: false},
		{name: "shell syntax", expansion: `!git status "$@"`, want: true},
		{name: "newline separator", expansion: "!git status --short\n/usr/bin/git clone source destination", want: true},
		{name: "target selection", expansion: "!/usr/bin/git -C ../other status", want: true},
		{name: "unknown git command", expansion: "!git custom-command", want: true},
		{name: "non git program", expansion: "!tool git status", want: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := shellAliasCreatesRepository(test.expansion); got != test.want {
				t.Fatalf("shellAliasCreatesRepository(%q) = %v, want %v", test.expansion, got, test.want)
			}
		})
	}
}

func TestSplitGitGlobalOptionsRecognizesSupportedTargetNeutralFlags(t *testing.T) {
	for _, option := range []string{
		"-p", "-P", "--no-pager", "--paginate", "--no-replace-objects",
		"--no-lazy-fetch", "--no-optional-locks", "--no-advice",
		"--literal-pathspecs", "--glob-pathspecs", "--noglob-pathspecs", "--icase-pathspecs",
	} {
		prefix, subcommand, remainder, ok := splitGitGlobalOptions([]string{option, "status", "--short"})
		if !ok || subcommand != "status" || !reflect.DeepEqual(prefix, []string{option}) || !reflect.DeepEqual(remainder, []string{"--short"}) {
			t.Fatalf("splitGitGlobalOptions(%q) = %#v, %q, %#v, %v", option, prefix, subcommand, remainder, ok)
		}
	}
}

func TestSplitGitGlobalOptionsRecognizesDocumentedValueFlags(t *testing.T) {
	for _, test := range []struct {
		args    []string
		prefix  []string
		command string
	}{
		{args: []string{"--attr-source", "HEAD", "check-attr", "--all", "--", "file"}, prefix: []string{"--attr-source", "HEAD"}, command: "check-attr"},
		{args: []string{"--attr-source=HEAD", "check-attr", "--all", "--", "file"}, prefix: []string{"--attr-source=HEAD"}, command: "check-attr"},
		{args: []string{"--list-cmds=builtins"}, prefix: []string{"--list-cmds=builtins"}},
		{args: []string{"--exec-path=/custom/git-core", "diff", "--cached"}, prefix: []string{"--exec-path=/custom/git-core"}, command: "diff"},
	} {
		prefix, subcommand, _, ok := splitGitGlobalOptions(test.args)
		if !ok || subcommand != test.command || !reflect.DeepEqual(prefix, test.prefix) {
			t.Fatalf("splitGitGlobalOptions(%#v) = %#v, %q, _, %v; want %#v, %q, _, true", test.args, prefix, subcommand, ok, test.prefix, test.command)
		}
	}
}

func TestPrivateIndexGitArgsDisablesSplitIndexAfterCallerConfiguration(t *testing.T) {
	got := privateIndexGitArgs([]string{"-c", "core.splitIndex=true", "status", "--short"})
	want := []string{"-c", "core.splitIndex=true", "-c", disableSplitIndexConfig, "status", "--short"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("privateIndexGitArgs() = %#v, want %#v", got, want)
	}
}
