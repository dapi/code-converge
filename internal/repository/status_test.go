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

func TestMain(m *testing.M) {
	if IsScopedGitWrapperInvocation(os.Args[0]) {
		os.Exit(RunScopedGitWrapper(os.Args[1:]))
	}
	os.Exit(m.Run())
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
	if err != nil || target.Source != "explicit" || target.Base != "develop" || target.MergeBase != "deadbeef" || len(target.Env) != 1 || !strings.HasPrefix(target.Env[0], "PATH=") {
		t.Fatalf("target=%#v err=%v", target, err)
	}
	if err := scope.Close(); err != nil {
		t.Fatal(err)
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
		case inv.Executable == "gh" && args == "pr list --repo "+fork+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"contributor/code-converge"}}]`}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+upstream+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository":
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
			return runner.Result{}, errors.New("origin is not configured")
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
		case inv.Executable == "git" && args == "merge-base HEAD refs/remotes/origin/main":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && args == "add -A":
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
		{"ssh://git@github.example.com/dapi/code-converge.git", "github.example.com/dapi/code-converge"},
		{"https://github.com/dapi/code-converge.git", "github.com/dapi/code-converge"},
		{"git@GitHub.COM:Dapi/Code-Converge.git", "github.com/dapi/code-converge"},
	} {
		if got, err := repositoryIdentity(test.remote); err != nil || got != test.want {
			t.Errorf("repositoryIdentity(%q) = %q, %v; want %q", test.remote, got, err, test.want)
		}
	}
	for _, remote := range []string{"file:///tmp/code-converge.git", "git@github.com:code-converge.git"} {
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
			return runner.Result{Stdout: "git@github.example.com:dapi/code-converge.git"}, nil
		case inv.Executable == "git" && args == "remote get-url --all enterprise":
			return runner.Result{Stdout: "git@github.example.com:dapi/code-converge.git"}, nil
		case inv.Executable == "gh" && args == "pr list --repo github.example.com/dapi/code-converge --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"dapi/code-converge"}}]`}, nil
		case inv.Executable == "git" && args == "remote":
			return runner.Result{Stdout: "enterprise"}, nil
		case inv.Executable == "git" && args == "rev-parse --verify refs/remotes/enterprise/main^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case inv.Executable == "git" && args == "merge-base HEAD refs/remotes/enterprise/main":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && args == "add -A":
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
		case inv.Executable == "gh" && args == "pr list --repo "+fork+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository":
			return runner.Result{Stdout: "[]"}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+upstream+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository":
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
		case inv.Executable == "gh" && args == "pr list --repo "+fork+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository":
			return runner.Result{Stdout: "[]"}, nil
		case inv.Executable == "gh" && args == "pr list --repo "+upstream+" --head feature --state open --json baseRefName,baseRefOid,headRefName,headRepository":
			return runner.Result{Stdout: `[{"baseRefName":"main","baseRefOid":"base-sha","headRefName":"feature","headRepository":{"nameWithOwner":"contributor/code-converge"}}]`}, nil
		case inv.Executable == "git" && args == "rev-parse --verify refs/remotes/upstream/main^{commit}":
			return runner.Result{Stdout: "base-sha"}, nil
		case inv.Executable == "git" && args == "merge-base HEAD refs/remotes/upstream/main":
			return runner.Result{Stdout: "merge-sha"}, nil
		case inv.Executable == "git" && args == "add -A":
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
				case inv.Executable == "git" && args == "merge-base HEAD main":
					return runner.Result{Stdout: "merge-sha"}, nil
				case inv.Executable == "git" && args == "add -A":
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
	// This descendant bypasses the wrapper's PATH entry. It inherits the
	// command-local index, so staging cannot overwrite the stable review index.
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
	if strings.Contains(string(nestedIndex), "absolute.txt") {
		t.Fatalf("absolute descendant used nested repository index: %q", nestedIndex)
	}
	afterAbsolute, err := os.ReadFile(privateIndex)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(afterAbsolute, before) {
		t.Fatal("absolute nested git operation changed the private review index")
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
	runGit("config", "alias.clone-repository", "clone")
	runGit("config", "alias.add-worktree", "worktree add")
	runGit("config", "alias.chained-worktree", "add-worktree")
	runGit("config", "alias.shell-worktree", `!f() { git worktree add "$@"; }; f`)
	runGit("config", "alias.review-status", "status --short")
	git := mustGitExecutable(t)

	for _, args := range [][]string{
		{"-C", root, "clone-repository", "source", "destination"},
		{"-C", root, "add-worktree", "destination"},
		{"-C", root, "chained-worktree", "destination"},
		{"-C", root, "shell-worktree", "destination"},
	} {
		if !gitCreatesRepository(args, git, root) {
			t.Fatalf("gitCreatesRepository(%#v) = false; want true", args)
		}
	}
	if gitCreatesRepository([]string{"-C", root, "review-status"}, git, root) {
		t.Fatal("gitCreatesRepository() classified a non-creating alias as repository creation")
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
