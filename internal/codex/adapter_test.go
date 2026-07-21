package codex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dapi/reviewer/internal/config"
	"github.com/dapi/reviewer/internal/runner"
)

func TestParseReview(t *testing.T) {
	tests := []struct {
		name    string
		report  string
		want    ReviewResult
		wantErr bool
	}{
		{"clean", "No findings.\n", ReviewResult{Clean: true}, false},
		{"clean heading", "## Review\nNo issues found.\n", ReviewResult{Clean: true}, false},
		{"priorities", "## Findings\n- [P0] critical\n- [P1] high\n- [P2] medium\n- [P3] low\n- [P4] future\n- no priority\n", ReviewResult{Counts: Counts{Critical: 1, High: 1, Medium: 1, Low: 1, Unknown: 2}}, false},
		{"inline finding", "[P1] unsafe change — file.go:12", ReviewResult{Counts: Counts{High: 1}}, false},
		{"empty", "", ReviewResult{}, true},
		{"ambiguous prose", "Looks mostly good to me.", ReviewResult{}, true},
		{"mixed", "## Findings\n- [P1] bug\nNo findings.", ReviewResult{}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseReview(test.report)
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
		`{"verdict":"FAILED","commit":"success","push":"success","change_request":"skipped","ci":"success"}`,
		`{"verdict":"FAILED","commit":"skipped","push":"skipped","change_request":"skipped","ci":"skipped"}`,
		`{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped","extra":true}`,
		`{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"} trailing`,
	}
	for _, data := range invalid {
		if _, err := ParseFinalization([]byte(data)); err == nil {
			t.Errorf("invalid result accepted: %s", data)
		}
	}
}

type recordingRunner struct {
	invocations []runner.Invocation
	result      runner.Result
	err         error
}

func (r *recordingRunner) Run(_ context.Context, invocation runner.Invocation) (runner.Result, error) {
	r.invocations = append(r.invocations, invocation)
	for index, arg := range invocation.Args {
		if arg == "--output-last-message" && index+1 < len(invocation.Args) && r.err == nil {
			_ = os.WriteFile(invocation.Args[index+1], []byte(`{"verdict":"SUCCESS","commit":"success","push":"success","change_request":"skipped","ci":"skipped"}`), 0o600)
		}
	}
	return r.result, r.err
}

func TestAdapterInvocations(t *testing.T) {
	r := &recordingRunner{result: runner.Result{Stdout: "No findings.\n"}}
	a := Adapter{Runner: r, Config: config.Config{
		ReviewModel: "review", ReviewEffort: "high", FixModel: "fix", FixEffort: "medium", FixPrompt: "fix it",
		FinalizeModel: "final", FinalizePrompt: "finalize", CIFixPrompt: "ci",
	}}
	if result, err := a.Review(context.Background()); err != nil || !result.Clean {
		t.Fatalf("review = %#v, %v", result, err)
	}
	if err := a.FixFindings(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Finalize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := a.FixCI(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.invocations[0].Args, " "); !strings.Contains(got, `model="review"`) || !strings.HasSuffix(got, "review --uncommitted") {
		t.Errorf("review args = %s", got)
	}
	if r.invocations[1].Stdin != "fix it" || r.invocations[3].Stdin != "ci" {
		t.Errorf("prompts not passed through: %#v", r.invocations)
	}
	finalArgs := strings.Join(r.invocations[2].Args, " ")
	if !strings.Contains(finalArgs, "--output-schema") || !strings.Contains(finalArgs, "--output-last-message") {
		t.Errorf("finalization args = %s", finalArgs)
	}
}

func TestAdapterPropagatesRunnerFailure(t *testing.T) {
	r := &recordingRunner{err: errors.New("boom")}
	a := Adapter{Runner: r, Config: config.Config{ReviewModel: "m", ReviewEffort: "e"}}
	if _, err := a.Review(context.Background()); err == nil {
		t.Fatal("expected runner error")
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
