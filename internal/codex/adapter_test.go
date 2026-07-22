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

	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/runner"
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
		{"clean findings heading", "## Findings\n- None found\n", ReviewResult{Clean: true}, false},
		{"priorities", "## Findings\n- [P0] critical\n- [P1] high\n- [P2] medium\n- [P3] low\n- [P4] future\n- no priority\n", ReviewResult{Counts: Counts{Critical: 1, High: 1, Medium: 1, Low: 1, Unknown: 1}}, false},
		{"nested explanatory bullet", "## Findings\n- [P1] bug\n  - explanatory detail\n", ReviewResult{Counts: Counts{High: 1}}, false},
		{"inline finding", "[P1] unsafe change — file.go:12", ReviewResult{Counts: Counts{High: 1}}, false},
		{"markdown finding prefixes", "### [P1] heading\n+ [P2] plus\n1. [P3] ordered\n", ReviewResult{Counts: Counts{Medium: 1, High: 1, Low: 1}}, false},
		{"headerless unsupported label", "[P1] bug\n[NOTE] context", ReviewResult{}, true},
		{"non-priority finding label", "## Findings\n- [NOTE] context\n", ReviewResult{}, true},
		{"non-priority label beside finding", "## Findings\n- [P1] bug\n- [NOTE] context\n", ReviewResult{}, true},
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

func TestParseStructuredReview(t *testing.T) {
	const clean = `{"findings":[],"overall_correctness":"patch is correct","overall_explanation":"No changes to review.","overall_confidence_score":0.99}`
	const findings = `{"findings":[{"title":"[P0] critical","body":"body","confidence_score":0.9,"priority":0,"code_location":{"absolute_file_path":"/tmp/a.go","line_range":{"start":1,"end":1}}},{"title":"[P1] high","body":"body","confidence_score":0.8,"priority":1,"code_location":{"absolute_file_path":"/tmp/b.go","line_range":{"start":2,"end":2}}},{"title":"[P2] medium","body":"body","confidence_score":0.7,"priority":2,"code_location":{"absolute_file_path":"/tmp/c.go","line_range":{"start":3,"end":3}}},{"title":"[P3] low","body":"body","confidence_score":0.6,"priority":3,"code_location":{"absolute_file_path":"/tmp/d.go","line_range":{"start":4,"end":4}}}],"overall_correctness":"patch is incorrect","overall_explanation":"findings","overall_confidence_score":0.8}`
	tests := []struct {
		name    string
		report  string
		want    ReviewResult
		wantErr bool
	}{
		{"clean", clean, ReviewResult{Clean: true}, false},
		{"priorities", findings, ReviewResult{Counts: Counts{Critical: 1, High: 1, Medium: 1, Low: 1}}, false},
		{"trailing", clean + " trailing", ReviewResult{}, true},
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
	const report = "## Findings\n- [P1] preserve this finding\n"
	r := &recordingRunner{result: runner.Result{Stdout: report}}
	a := Adapter{Runner: r, Config: config.Config{
		ReviewModel: "gpt-5.6-sol", ReviewEffort: "high", FixModel: "gpt-5.6-terra", FixEffort: "high", FixPrompt: "fix it",
		FinalizeModel: "gpt-5.6-luna", FinalizeEffort: "medium", FinalizePrompt: "finalize",
		CIFixModel: "gpt-5.6-terra", CIFixEffort: "high", CIFixPrompt: "ci",
	}}
	if result, err := a.Review(context.Background()); err != nil || result.Clean || result.Report != strings.TrimSpace(report) {
		t.Fatalf("review = %#v, %v", result, err)
	}
	if err := a.FixFindings(context.Background(), strings.TrimSpace(report)); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Finalize(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := a.FixCI(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantPairs := []struct{ model, effort string }{{"gpt-5.6-sol", "high"}, {"gpt-5.6-terra", "high"}, {"gpt-5.6-luna", "medium"}, {"gpt-5.6-terra", "high"}}
	for i, want := range wantPairs {
		got := strings.Join(r.invocations[i].Args, " ")
		if !strings.Contains(got, `model="`+want.model+`"`) || !strings.Contains(got, `model_reasoning_effort="`+want.effort+`"`) {
			t.Errorf("invocation %d args = %s", i, got)
		}
	}
	if got := strings.Join(r.invocations[0].Args, " "); !strings.HasSuffix(got, "review --uncommitted") {
		t.Errorf("review args = %s", got)
	}
	if r.invocations[1].Stdin != "fix it\n\nReview findings to address:\n\n"+strings.TrimSpace(report) || r.invocations[3].Stdin != "ci" {
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

func TestCountsTotal(t *testing.T) {
	counts := Counts{Critical: 1, High: 2, Medium: 3, Low: 4, Unknown: 5}
	if got := counts.Total(); got != 15 {
		t.Fatalf("Total() = %d, want 15", got)
	}
}

func TestReviewWithUnclassifiableReport(t *testing.T) {
	r := &recordingRunner{result: runner.Result{Stdout: "Looks mostly good to me."}}
	a := Adapter{Runner: r, Config: config.Config{ReviewModel: "m", ReviewEffort: "e"}}
	_, err := a.Review(context.Background())
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestReviewWithStructuredFindingsPreservesReport(t *testing.T) {
	const report = "{\"findings\":[{\"title\":\"[P1] high\",\"body\":\"body\",\"confidence_score\":0.9,\"priority\":1,\"code_location\":{\"absolute_file_path\":\"/tmp/a.go\",\"line_range\":{\"start\":1,\"end\":1}}}],\"overall_correctness\":\"patch is incorrect\",\"overall_explanation\":\"finding\",\"overall_confidence_score\":0.9}\n"
	r := &recordingRunner{result: runner.Result{Stdout: report}}
	a := Adapter{Runner: r, Config: config.Config{ReviewModel: "m", ReviewEffort: "e"}}
	result, err := a.Review(context.Background())
	if err != nil || result.Clean || result.Counts != (Counts{High: 1}) || result.Report != strings.TrimSpace(report) {
		t.Fatalf("review = %#v, %v", result, err)
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
