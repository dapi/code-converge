package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/repository"
	"github.com/dapi/code-converge/internal/runner"
)

const (
	cleanReviewJSON    = `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"No changes to review.","overall_confidence_score":0.99}`
	findingsReviewJSON = `{"findings":[{"title":"[P0] critical","body":"body","confidence_score":0.9,"priority":0,"code_location":{"absolute_file_path":"/tmp/a.go","line_range":{"start":1,"end":1}}},{"title":"[P1] high","body":"body","confidence_score":0.8,"priority":1,"code_location":{"absolute_file_path":"/tmp/b.go","line_range":{"start":2,"end":2}}},{"title":"[P2] medium","body":"body","confidence_score":0.7,"priority":2,"code_location":{"absolute_file_path":"/tmp/c.go","line_range":{"start":3,"end":3}}},{"title":"[P3] low","body":"body","confidence_score":0.6,"priority":3,"code_location":{"absolute_file_path":"/tmp/d.go","line_range":{"start":4,"end":4}}}],"overall_correctness":"patch is incorrect","overall_explanation":"findings","overall_confidence_score":0.8}`
	finalizationJSON   = `{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`
)

func TestParseStructuredReview(t *testing.T) {
	tests := []struct {
		name    string
		report  string
		want    ReviewResult
		wantErr bool
	}{
		{"clean", cleanReviewJSON, ReviewResult{Clean: true}, false},
		{"priorities", findingsReviewJSON, ReviewResult{Counts: Counts{Critical: 1, High: 1, Medium: 1, Low: 1}}, false},
		{"trailing", cleanReviewJSON + " trailing", ReviewResult{}, true},
		{"unknown top-level field", `{"findings":[],"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1,"extra":true}`, ReviewResult{}, true},
		{"case-variant top-level field", `{"Findings":[],"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1}`, ReviewResult{}, true},
		{"case-variant duplicate top-level field", `{"findings":[],"Findings":[],"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1}`, ReviewResult{}, true},
		{"missing field", `{"findings":[],"overall_correctness":"ok","overall_explanation":"ok"}`, ReviewResult{}, true},
		{"wrong field type", `{"findings":{},"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1}`, ReviewResult{}, true},
		{"duplicate nested field", `{"findings":[{"title":"a","title":"b","body":"body","confidence_score":1,"priority":1,"code_location":{"absolute_file_path":"a","line_range":{"start":1,"end":1}}}],"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1}`, ReviewResult{}, true},
		{"unknown finding field", `{"findings":[{"title":"a","body":"body","confidence_score":1,"priority":1,"code_location":{"absolute_file_path":"a","line_range":{"start":1,"end":1}},"extra":true}],"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1}`, ReviewResult{}, true},
		{"invalid priority", `{"findings":[{"title":"a","body":"body","confidence_score":1,"priority":4,"code_location":{"absolute_file_path":"a","line_range":{"start":1,"end":1}}}],"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1}`, ReviewResult{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseStructuredReview([]byte(test.report))
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("result = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseFinalization(t *testing.T) {
	valid := []Finalization{
		{Verdict: "SUCCESS", Commit: "success", Push: "success", ChangeRequest: "success", CI: "success"},
		{Verdict: "SUCCESS", Commit: "success", Push: "success", ChangeRequest: "skipped", CI: "skipped"},
		{Verdict: "SUCCESS", Commit: "skipped", Push: "skipped", ChangeRequest: "skipped", CI: "skipped"},
		{Verdict: "CI_FAILED", Commit: "success", Push: "success", ChangeRequest: "skipped", CI: "failed"},
		{Verdict: "FAILED", Commit: "failed", Push: "skipped", ChangeRequest: "skipped", CI: "skipped"},
	}
	for _, value := range valid {
		data, _ := json.Marshal(value)
		if _, err := ParseFinalization(data); err != nil {
			t.Errorf("valid result %#v rejected: %v", value, err)
		}
	}
	invalid := []string{
		`{}`,
		`{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"failed"}`,
		`{"verdict":"CI_FAILED","commit":"success","push":"success","change_request":"skipped","ci":"success"}`,
		`{"verdict":"FAILED","commit":"success","push":"success","change_request":"skipped","ci":"failed"}`,
		`{"verdict":"FAILED","commit":"success","push":"success","change_request":"skipped","ci":"success"}`,
		`{"verdict":"FAILED","commit":"skipped","push":"skipped","change_request":"skipped","ci":"skipped"}`,
		`{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped","extra":true}`,
		`{"verdict":"FAILED","verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`,
		`{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"} trailing`,
	}
	for _, data := range invalid {
		if _, err := ParseFinalization([]byte(data)); err == nil {
			t.Errorf("invalid result accepted: %s", data)
		}
	}
}

type recordingRunner struct {
	invocations       []runner.Invocation
	codexResult       runner.Result
	codexErr          error
	reviewMessage     []byte
	writeReview       bool
	baseCommit        string
	mergeBase         string
	reviewSchema      []byte
	reviewSchemaPath  string
	reviewMessagePath string
	reviewDirMode     os.FileMode
	reviewSchemaMode  os.FileMode
}

func (r *recordingRunner) Run(_ context.Context, invocation runner.Invocation) (runner.Result, error) {
	r.invocations = append(r.invocations, invocation)
	if invocation.Executable == "git" {
		switch strings.Join(invocation.Args, " ") {
		case "rev-parse --verify base^{commit}":
			if r.baseCommit == "" {
				return runner.Result{Stdout: "1111111111111111111111111111111111111111"}, nil
			}
			return runner.Result{Stdout: r.baseCommit}, nil
		case "merge-base HEAD base":
			if r.mergeBase == "" {
				return runner.Result{Stdout: "2222222222222222222222222222222222222222"}, nil
			}
			return runner.Result{Stdout: r.mergeBase}, nil
		case "rev-parse --git-path index":
			return runner.Result{Stdout: ".git/index"}, nil
		case "add -A":
			return runner.Result{}, nil
		default:
			return runner.Result{}, errors.New("unexpected git invocation")
		}
	}
	for index, arg := range invocation.Args {
		if arg != "--output-last-message" || index+1 >= len(invocation.Args) {
			continue
		}
		messagePath := invocation.Args[index+1]
		if strings.Contains(invocation.Stdin, "prepared private Git index") {
			r.reviewMessagePath = messagePath
			for schemaIndex, schemaArg := range invocation.Args {
				if schemaArg == "--output-schema" && schemaIndex+1 < len(invocation.Args) {
					r.reviewSchemaPath = invocation.Args[schemaIndex+1]
					r.reviewSchema, _ = os.ReadFile(r.reviewSchemaPath)
					if info, err := os.Stat(r.reviewSchemaPath); err == nil {
						r.reviewSchemaMode = info.Mode().Perm()
					}
					if info, err := os.Stat(filepath.Dir(r.reviewSchemaPath)); err == nil {
						r.reviewDirMode = info.Mode().Perm()
					}
				}
			}
			if r.writeReview {
				_ = os.WriteFile(messagePath, r.reviewMessage, 0o600)
			}
		} else {
			_ = os.WriteFile(messagePath, []byte(finalizationJSON), 0o600)
		}
	}
	return r.codexResult, r.codexErr
}

func newReviewAdapter(t *testing.T, r *recordingRunner, cfg config.Config) Adapter {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".git", "index"), []byte("test index"), 0o600); err != nil {
		t.Fatal(err)
	}
	scope := &repository.ReviewScope{Runner: r, Base: "base", Root: root}
	t.Cleanup(func() {
		if err := scope.Close(); err != nil {
			t.Errorf("close review scope: %v", err)
		}
	})
	return Adapter{Runner: r, Config: cfg, ReviewScope: scope}
}

func codexInvocations(invocations []runner.Invocation) []runner.Invocation {
	var result []runner.Invocation
	for _, invocation := range invocations {
		if invocation.Executable == "" {
			result = append(result, invocation)
		}
	}
	return result
}

func TestAdapterInvocations(t *testing.T) {
	r := &recordingRunner{
		codexResult:   runner.Result{Stdout: cleanReviewJSON, Stderr: findingsReviewJSON},
		reviewMessage: []byte(findingsReviewJSON + "\n"),
		writeReview:   true,
	}
	a := newReviewAdapter(t, r, config.Config{
		ReviewModel: "gpt-5.6-sol", ReviewEffort: "high", FixModel: "gpt-5.6-terra", FixEffort: "high", FixPrompt: "fix it",
		FinalizeModel: "gpt-5.6-luna", FinalizeEffort: "medium", FinalizePrompt: "finalize",
		CIFixModel: "gpt-5.6-terra", CIFixEffort: "high", CIFixPrompt: "ci",
	})
	result, err := a.Review(context.Background())
	if err != nil || result.Clean || result.Counts != (Counts{Critical: 1, High: 1, Medium: 1, Low: 1}) || result.Report != findingsReviewJSON {
		t.Fatalf("review = %#v, %v", result, err)
	}
	if err := a.FixFindings(context.Background(), result.Report); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Finalize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := a.FixCI(context.Background()); err != nil {
		t.Fatal(err)
	}
	invocations := codexInvocations(r.invocations)
	wantPairs := []struct{ model, effort string }{{"gpt-5.6-sol", "high"}, {"gpt-5.6-terra", "high"}, {"gpt-5.6-luna", "medium"}, {"gpt-5.6-terra", "high"}}
	for i, want := range wantPairs {
		got := strings.Join(invocations[i].Args, " ")
		if !strings.Contains(got, `model="`+want.model+`"`) || !strings.Contains(got, `model_reasoning_effort="`+want.effort+`"`) {
			t.Errorf("invocation %d args = %s", i, got)
		}
	}
	if got := strings.Join(invocations[0].Args, " "); !strings.Contains(got, " exec --output-schema ") || !strings.Contains(got, " --output-last-message ") || !strings.HasSuffix(got, " -") {
		t.Errorf("review args = %s", got)
	}
	if invocations[1].Stdin != "fix it\n\nReview findings to address:\n\n"+findingsReviewJSON || invocations[3].Stdin != "ci" {
		t.Errorf("prompts not passed through: %#v", invocations)
	}
	finalArgs := strings.Join(invocations[2].Args, " ")
	if !strings.Contains(finalArgs, "--output-schema") || !strings.Contains(finalArgs, "--output-last-message") {
		t.Errorf("finalization args = %s", finalArgs)
	}
	if r.reviewDirMode != 0o700 || r.reviewSchemaMode != 0o600 {
		t.Errorf("review workspace modes = dir %o schema %o", r.reviewDirMode, r.reviewSchemaMode)
	}
	if _, err := os.Stat(r.reviewSchemaPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("review schema was not cleaned up: %v", err)
	}
	if _, err := os.Stat(r.reviewMessagePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("review message was not cleaned up: %v", err)
	}
}

func TestAdapterPropagatesRunnerFailure(t *testing.T) {
	r := &recordingRunner{
		codexErr:      errors.New("boom"),
		reviewMessage: []byte(cleanReviewJSON),
		writeReview:   true,
	}
	a := newReviewAdapter(t, r, config.Config{ReviewModel: "m", ReviewEffort: "e"})
	if _, err := a.Review(context.Background()); err == nil || err.Error() != "boom" {
		t.Fatalf("error = %v", err)
	}
}

func TestFinalizationSchemaIsStrictJSON(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(finalizationSchema), &schema); err != nil {
		t.Fatal(err)
	}
	if schema["additionalProperties"] != false {
		t.Fatalf("schema is not strict: %#v", schema)
	}
	if filepath.Ext("schema.json") != ".json" { // keep filepath import exercised on every supported OS
		t.Fatal("unexpected filepath behavior")
	}
}

func TestReviewSchemaIsStrictJSON(t *testing.T) {
	var schema map[string]any
	if err := json.Unmarshal([]byte(reviewSchema), &schema); err != nil {
		t.Fatal(err)
	}
	if schema["additionalProperties"] != false {
		t.Fatalf("top-level schema is not strict: %#v", schema)
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties = %#v", schema["properties"])
	}
	findings, ok := properties["findings"].(map[string]any)
	if !ok {
		t.Fatalf("findings schema = %#v", properties["findings"])
	}
	finding, ok := findings["items"].(map[string]any)
	if !ok || finding["additionalProperties"] != false {
		t.Fatalf("finding schema is not strict: %#v", findings["items"])
	}
	findingProperties, ok := finding["properties"].(map[string]any)
	if !ok {
		t.Fatalf("finding properties = %#v", finding["properties"])
	}
	priority, ok := findingProperties["priority"].(map[string]any)
	if !ok || !reflect.DeepEqual(priority["enum"], []any{float64(0), float64(1), float64(2), float64(3)}) {
		t.Fatalf("priority schema = %#v", findingProperties["priority"])
	}
	location, ok := findingProperties["code_location"].(map[string]any)
	if !ok || location["additionalProperties"] != false {
		t.Fatalf("location schema is not strict: %#v", findingProperties["code_location"])
	}
	locationProperties, ok := location["properties"].(map[string]any)
	if !ok {
		t.Fatalf("location properties = %#v", location["properties"])
	}
	lineRange, ok := locationProperties["line_range"].(map[string]any)
	if !ok || lineRange["additionalProperties"] != false {
		t.Fatalf("line range schema is not strict: %#v", locationProperties["line_range"])
	}
}

func TestCountsTotal(t *testing.T) {
	counts := Counts{Critical: 1, High: 2, Medium: 3, Low: 4, Unknown: 5}
	if got := counts.Total(); got != 15 {
		t.Fatalf("Total() = %d, want 15", got)
	}
}

func TestReviewRequiresScope(t *testing.T) {
	r := &recordingRunner{}
	a := Adapter{Runner: r, Config: config.Config{ReviewModel: "m", ReviewEffort: "e"}}
	if _, err := a.Review(context.Background()); err == nil || !strings.Contains(err.Error(), "review scope is required") {
		t.Fatalf("error = %v", err)
	}
	if len(r.invocations) != 0 {
		t.Fatalf("unexpected invocations: %#v", r.invocations)
	}
}

func TestReviewRejectsInvalidFinalResponse(t *testing.T) {
	tests := []struct {
		name      string
		message   []byte
		writeFile bool
		want      string
	}{
		{name: "missing", writeFile: false, want: "read review response"},
		{name: "empty", message: []byte{}, writeFile: true, want: "parse structured review response"},
		{name: "prose", message: []byte("No findings."), writeFile: true, want: "parse structured review response"},
		{name: "malformed", message: []byte("{"), writeFile: true, want: "parse structured review response"},
		{name: "incomplete", message: []byte(`{"findings":[]}`), writeFile: true, want: "contains unknown or missing fields"},
		{name: "unknown field", message: []byte(`{"findings":[],"overall_correctness":"ok","overall_explanation":"ok","overall_confidence_score":1,"extra":true}`), writeFile: true, want: "contains unknown or missing fields"},
		{name: "invalid priority", message: []byte(`{"findings":[{"title":"a","body":"b","confidence_score":1,"priority":4,"code_location":{"absolute_file_path":"/a","line_range":{"start":1,"end":1}}}],"overall_correctness":"bad","overall_explanation":"bad","overall_confidence_score":1}`), writeFile: true, want: "invalid finding priority"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := &recordingRunner{
				codexResult:   runner.Result{Stdout: cleanReviewJSON, Stderr: cleanReviewJSON},
				reviewMessage: test.message,
				writeReview:   test.writeFile,
			}
			a := newReviewAdapter(t, r, config.Config{ReviewModel: "m", ReviewEffort: "e"})
			if _, err := a.Review(context.Background()); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestReviewUsesOnlyFinalResponseAndPreservesTarget(t *testing.T) {
	r := &recordingRunner{
		codexResult:   runner.Result{Stdout: findingsReviewJSON, Stderr: "No findings."},
		reviewMessage: []byte(cleanReviewJSON + "\n"),
		writeReview:   true,
		baseCommit:    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		mergeBase:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	}
	a := newReviewAdapter(t, r, config.Config{ReviewModel: "review-model", ReviewEffort: "high"})
	result, err := a.Review(context.Background())
	if err != nil || !result.Clean || result.Counts.Total() != 0 || result.Report != cleanReviewJSON {
		t.Fatalf("review = %#v, %v", result, err)
	}
	invocations := codexInvocations(r.invocations)
	if len(invocations) != 1 {
		t.Fatalf("codex invocations = %#v", invocations)
	}
	invocation := invocations[0]
	if len(invocation.Env) != 1 || !strings.HasPrefix(invocation.Env[0], "GIT_INDEX_FILE=") {
		t.Fatalf("review env = %#v", invocation.Env)
	}
	indexPath := strings.TrimPrefix(invocation.Env[0], "GIT_INDEX_FILE=")
	args := strings.Join(invocation.Args, " ")
	override := "shell_environment_policy.set.GIT_INDEX_FILE=" + strconv.Quote(indexPath)
	for _, want := range []string{"exec", "--output-schema", "--output-last-message", override} {
		if !strings.Contains(args, want) {
			t.Errorf("review args %q do not contain %q", args, want)
		}
	}
	for _, want := range []string{r.baseCommit, r.mergeBase, indexPath, "git diff --cached " + r.mergeBase} {
		if !strings.Contains(invocation.Stdin, want) {
			t.Errorf("review prompt does not contain %q:\n%s", want, invocation.Stdin)
		}
	}
	if result.Scope.BaseCommit != r.baseCommit || result.Scope.MergeBase != r.mergeBase || !reflect.DeepEqual(result.Scope.Env, invocation.Env) {
		t.Fatalf("scope = %#v, invocation env = %#v", result.Scope, invocation.Env)
	}
}

func TestReviewTargetValidation(t *testing.T) {
	t.Run("index environment", func(t *testing.T) {
		if got, err := reviewIndexPath([]string{"OTHER=value", "GIT_INDEX_FILE=/tmp/a=b"}); err != nil || got != "/tmp/a=b" {
			t.Fatalf("index path = %q, %v", got, err)
		}
		for _, test := range []struct {
			name string
			env  []string
		}{
			{name: "missing", env: nil},
			{name: "empty", env: []string{"GIT_INDEX_FILE="}},
			{name: "whitespace", env: []string{"GIT_INDEX_FILE= "}},
			{name: "duplicate", env: []string{"GIT_INDEX_FILE=/a", "GIT_INDEX_FILE=/b"}},
			{name: "missing equals", env: []string{"GIT_INDEX_FILE"}},
		} {
			t.Run(test.name, func(t *testing.T) {
				if _, err := reviewIndexPath(test.env); err == nil {
					t.Fatal("expected target validation error")
				}
			})
		}
	})
	for _, test := range []struct {
		name       string
		baseCommit string
		mergeBase  string
	}{
		{name: "missing base", baseCommit: " ", mergeBase: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{name: "missing merge base", baseCommit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", mergeBase: " "},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &recordingRunner{baseCommit: test.baseCommit, mergeBase: test.mergeBase}
			a := newReviewAdapter(t, r, config.Config{ReviewModel: "m"})
			if _, err := a.Review(context.Background()); err == nil || !strings.Contains(err.Error(), "selected base commit and merge base") {
				t.Fatalf("error = %v", err)
			}
			if len(codexInvocations(r.invocations)) != 0 {
				t.Fatalf("Codex invoked with invalid target: %#v", r.invocations)
			}
		})
	}
}

func TestReviewFakeExecutableBoundary(t *testing.T) {
	root := t.TempDir()
	runGit := func(args ...string) string {
		t.Helper()
		command := exec.Command("git", append([]string{"-C", root}, args...)...)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, output)
		}
		return strings.TrimSpace(string(output))
	}
	runGit("init", "-q")
	runGit("config", "user.name", "Test")
	runGit("config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(root, "review.txt"), []byte("base\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit("add", "review.txt")
	runGit("commit", "-qm", "base")
	baseCommit := runGit("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(root, "review.txt"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	fixtureDir := t.TempDir()
	scriptPath := filepath.Join(fixtureDir, "fake-codex")
	script := `#!/bin/sh
set -eu
printf '%s\n' "$@" > "$FT022_CAPTURE_DIR/args"
pwd > "$FT022_CAPTURE_DIR/cwd"
printf '%s' "${GIT_INDEX_FILE-}" > "$FT022_CAPTURE_DIR/index"
cat > "$FT022_CAPTURE_DIR/stdin"
schema=
message=
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output-schema)
      schema=$2
      shift 2
      ;;
    --output-last-message)
      message=$2
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
cp "$schema" "$FT022_CAPTURE_DIR/schema"
cp "$FT022_RESPONSE_FILE" "$message"
printf 'untrusted stdout'
printf 'untrusted stderr' >&2
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	responsePath := filepath.Join(fixtureDir, "response.json")
	if err := os.WriteFile(responsePath, []byte(cleanReviewJSON), 0o600); err != nil {
		t.Fatal(err)
	}
	captureDir := filepath.Join(fixtureDir, "capture")
	if err := os.Mkdir(captureDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FT022_CAPTURE_DIR", captureDir)
	t.Setenv("FT022_RESPONSE_FILE", responsePath)

	execRunner := runner.Exec{Executable: scriptPath, Dir: root}
	scope := &repository.ReviewScope{Runner: execRunner, Base: "HEAD", Root: root}
	t.Cleanup(func() {
		if err := scope.Close(); err != nil {
			t.Errorf("close review scope: %v", err)
		}
	})
	adapter := Adapter{
		Runner:      execRunner,
		ReviewScope: scope,
		Config:      config.Config{ReviewModel: "review-model", ReviewEffort: "high"},
	}
	result, err := adapter.Review(context.Background())
	if err != nil || !result.Clean || result.Report != cleanReviewJSON {
		t.Fatalf("review = %#v, %v", result, err)
	}

	read := func(name string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(captureDir, name))
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(data))
	}
	gotCWD, err := filepath.EvalSymlinks(read("cwd"))
	if err != nil {
		t.Fatal(err)
	}
	wantCWD, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if gotCWD != wantCWD {
		t.Errorf("cwd = %q, want %q", gotCWD, wantCWD)
	}
	indexPath := read("index")
	if indexPath == "" || !reflect.DeepEqual(result.Scope.Env, []string{"GIT_INDEX_FILE=" + indexPath}) {
		t.Fatalf("index = %q scope = %#v", indexPath, result.Scope)
	}
	args := read("args")
	for _, want := range []string{
		"exec",
		"--output-schema",
		"--output-last-message",
		"shell_environment_policy.set.GIT_INDEX_FILE=" + strconv.Quote(indexPath),
	} {
		if !strings.Contains(args, want) {
			t.Errorf("args do not contain %q:\n%s", want, args)
		}
	}
	prompt := read("stdin")
	for _, want := range []string{baseCommit, indexPath, "git diff --cached " + baseCommit} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt does not contain %q:\n%s", want, prompt)
		}
	}
	var gotSchema, wantSchema map[string]any
	if err := json.Unmarshal([]byte(read("schema")), &gotSchema); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(reviewSchema), &wantSchema); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotSchema, wantSchema) {
		t.Errorf("fake executable received unexpected schema")
	}
}

type codexFakeRunner struct {
	result     runner.Result
	err        error
	writePath  string
	writeBytes []byte
	writeFile  bool
}

func (r *codexFakeRunner) Run(_ context.Context, invocation runner.Invocation) (runner.Result, error) {
	for i, arg := range invocation.Args {
		if arg == "--output-last-message" && i+1 < len(invocation.Args) && r.writeFile {
			_ = os.WriteFile(invocation.Args[i+1], r.writeBytes, 0o600)
		}
	}
	return r.result, r.err
}

func TestFixCIWithModel(t *testing.T) {
	r := &recordingRunner{}
	a := Adapter{Runner: r, Config: config.Config{CIFixModel: "ci-model", CIFixEffort: "high", CIFixPrompt: "fix ci"}}
	if err := a.FixCI(context.Background()); err != nil {
		t.Fatal(err)
	}
	args := strings.Join(r.invocations[0].Args, " ")
	if !strings.Contains(args, `model="ci-model"`) || !strings.Contains(args, `model_reasoning_effort="high"`) {
		t.Fatalf("missing ci model/effort in args: %s", args)
	}
}

func TestFinalizeReadMessageError(t *testing.T) {
	r := &codexFakeRunner{result: runner.Result{}, writeFile: false}
	a := Adapter{Runner: r, Config: config.Config{FinalizeModel: "m", FinalizePrompt: "p"}}
	_, err := a.Finalize(context.Background())
	if err == nil || !strings.Contains(err.Error(), "read finalization response") {
		t.Fatalf("error = %v", err)
	}
}

func TestFinalizeParseError(t *testing.T) {
	r := &codexFakeRunner{result: runner.Result{}, writeFile: true, writeBytes: []byte(`not json`)}
	a := Adapter{Runner: r, Config: config.Config{FinalizeModel: "m", FinalizePrompt: "p"}}
	_, err := a.Finalize(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestRejectDuplicateJSONKeysNestedCases(t *testing.T) {
	valid := []string{
		`{"verdict":"SUCCESS","nested":{"a":1,"b":[1,2,{"c":3}]}}`,
		`{"verdict":"SUCCESS","list":[{"a":1},{"a":2}]}`,
	}
	for _, data := range valid {
		if err := rejectDuplicateJSONKeys([]byte(data)); err != nil {
			t.Errorf("valid data rejected: %s: %v", data, err)
		}
	}
	invalid := []string{
		`{"verdict":"SUCCESS","nested":{"a":1,"a":2}}`,
		`{"verdict":"SUCCESS","list":[{"a":1,"a":2}]}`,
		`{"verdict":"SUCCESS","a":{"b":1},"a":2}`,
		`{"verdict":"SUCCESS"} trailing`,
	}
	for _, data := range invalid {
		if err := rejectDuplicateJSONKeys([]byte(data)); err == nil {
			t.Errorf("invalid data accepted: %s", data)
		}
	}
}

func TestValidateFinalizationEdgeCases(t *testing.T) {
	invalid := []Finalization{
		{Verdict: "SUCCESS", Commit: "success", Push: "success", ChangeRequest: "success", CI: "ok"},
		{Verdict: "UNKNOWN", Commit: "success", Push: "success", ChangeRequest: "success", CI: "success"},
		{Verdict: "FAILED", Commit: "success", Push: "success", ChangeRequest: "skipped", CI: "success"},
		{Verdict: "FAILED", Commit: "success", Push: "success", ChangeRequest: "skipped", CI: "failed"},
		{Verdict: "FAILED", Commit: "skipped", Push: "skipped", ChangeRequest: "skipped", CI: "skipped"},
	}
	for _, value := range invalid {
		data, _ := json.Marshal(value)
		if _, err := ParseFinalization(data); err == nil {
			t.Errorf("invalid result accepted: %#v", value)
		}
	}
}
