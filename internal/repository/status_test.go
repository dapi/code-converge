package repository

import (
	"context"
	"errors"
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
	scope := &ReviewScope{Runner: fake, Base: "develop"}
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
			return runner.Result{Stdout: `[{"baseRefName":"develop"}]`}, nil
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
	target, err := (&ReviewScope{Runner: fake}).Prepare(context.Background())
	if err != nil || target.Source != "open_pr" || target.Base != "refs/remotes/origin/develop" {
		t.Fatalf("target=%#v err=%v", target, err)
	}
}
