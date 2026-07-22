package repository

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/dapi/code-converge/internal/runner"
)

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
