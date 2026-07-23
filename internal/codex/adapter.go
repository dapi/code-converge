package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dapi/code-converge/internal/config"
	"github.com/dapi/code-converge/internal/repository"
	"github.com/dapi/code-converge/internal/runner"
)

type Counts struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Unknown  int
}

func (c Counts) Total() int { return c.Critical + c.High + c.Medium + c.Low + c.Unknown }

type ReviewResult struct {
	Clean  bool
	Counts Counts
	Report string
	Scope  repository.ReviewTarget
}

type structuredReview struct {
	Findings               *[]structuredFinding `json:"findings"`
	OverallCorrectness     *string              `json:"overall_correctness"`
	OverallExplanation     *string              `json:"overall_explanation"`
	OverallConfidenceScore *float64             `json:"overall_confidence_score"`
}

type structuredFinding struct {
	Title           *string                 `json:"title"`
	Body            *string                 `json:"body"`
	ConfidenceScore *float64                `json:"confidence_score"`
	Priority        *int                    `json:"priority"`
	CodeLocation    *structuredCodeLocation `json:"code_location"`
}

type structuredCodeLocation struct {
	AbsoluteFilePath *string              `json:"absolute_file_path"`
	LineRange        *structuredLineRange `json:"line_range"`
}

type structuredLineRange struct {
	Start *int `json:"start"`
	End   *int `json:"end"`
}

type Finalization struct {
	Verdict       string `json:"verdict"`
	Commit        string `json:"commit"`
	Push          string `json:"push"`
	ChangeRequest string `json:"change_request"`
	CI            string `json:"ci"`
}

type Adapter struct {
	Runner      runner.Runner
	Config      config.Config
	ReviewScope *repository.ReviewScope
	Output      func(source string, data []byte)
}

func (a Adapter) Review(ctx context.Context) (ReviewResult, error) {
	if a.ReviewScope == nil {
		return ReviewResult{}, errors.New("review scope is required")
	}
	target, err := a.ReviewScope.Prepare(ctx)
	if err != nil {
		return ReviewResult{}, err
	}
	if strings.TrimSpace(target.BaseCommit) == "" || strings.TrimSpace(target.MergeBase) == "" {
		return ReviewResult{}, errors.New("review target requires a selected base commit and merge base")
	}
	args, err := scopedReviewArgs(a.Config, target)
	if err != nil {
		return ReviewResult{}, err
	}

	dir, err := os.MkdirTemp("", "code-converge-review-response-")
	if err != nil {
		return ReviewResult{}, fmt.Errorf("create review response workspace: %w", err)
	}
	defer os.RemoveAll(dir)
	schemaPath := filepath.Join(dir, "schema.json")
	messagePath := filepath.Join(dir, "message.json")
	if err := os.WriteFile(schemaPath, []byte(reviewSchema), 0o600); err != nil {
		return ReviewResult{}, fmt.Errorf("write review output schema: %w", err)
	}

	args = append(
		args,
		"exec", "--output-schema", schemaPath, "--output-last-message", messagePath, "-",
	)
	if _, err := a.Runner.Run(ctx, runner.Invocation{
		Args:   args,
		Env:    target.Env,
		Stdin:  reviewPrompt(target),
		Output: a.output(),
	}); err != nil {
		return ReviewResult{}, err
	}
	message, err := os.ReadFile(messagePath)
	if err != nil {
		return ReviewResult{}, fmt.Errorf("read review response: %w", err)
	}
	review, err := parseStructuredReview(message)
	if err != nil {
		return ReviewResult{}, err
	}
	review.Report = strings.TrimSpace(string(message))
	review.Scope = target
	return review, nil
}

func scopedReviewArgs(configuration config.Config, target repository.ReviewTarget) ([]string, error) {
	path, ok := environmentValue(target.Env, "PATH")
	if !ok || path == "" {
		return nil, errors.New("review scope did not provide wrapper PATH")
	}
	args := modelArgs(configuration.ReviewModel, configuration.ReviewEffort)
	// A login shell can prepend a system Git directory after Codex applies PATH.
	// Disable login-shell startup for the review so its wrapper remains first
	// without exporting GIT_INDEX_FILE beyond the scoped Git helper.
	args = append(args, "-c", "shell_environment_policy.set.PATH="+strconv.Quote(path))
	args = append(args, "-c", "allow_login_shell=false")
	return args, nil
}

func environmentValue(environment []string, name string) (string, bool) {
	prefix := name + "="
	for index := len(environment) - 1; index >= 0; index-- {
		if strings.HasPrefix(environment[index], prefix) {
			return strings.TrimPrefix(environment[index], prefix), true
		}
	}
	return "", false
}

func reviewPrompt(target repository.ReviewTarget) string {
	return fmt.Sprintf(
		`Review the changes in the prepared private Git index.

Selected base commit: %s
Merge base and comparison start: %s

A scoped Git helper exposes the private snapshot only to Git commands that target the reviewed repository. Review the equivalent of git diff --cached %s so the comparison covers the merge-base-to-private-snapshot change. Inspect related files when needed, but do not modify the repository, the real index, or the worktree.

Return actionable code-review findings. Use an empty findings array when there are none. Return only the JSON object required by the supplied output schema.`,
		target.BaseCommit,
		target.MergeBase,
		target.MergeBase,
	)
}

func (a Adapter) FixFindings(ctx context.Context, report string) error {
	prompt := a.Config.FixPrompt + "\n\nReview findings to address:\n\n" + report
	_, err := a.Runner.Run(ctx, runner.Invocation{Args: append(modelArgs(a.Config.FixModel, a.Config.FixEffort), "exec", "-"), Stdin: prompt, Output: a.output()})
	return err
}

func (a Adapter) FixCI(ctx context.Context) error {
	args := append(modelArgs(a.Config.CIFixModel, a.Config.CIFixEffort), "exec", "-")
	_, err := a.Runner.Run(ctx, runner.Invocation{Args: args, Stdin: a.Config.CIFixPrompt, Output: a.output()})
	return err
}

func (a Adapter) Finalize(ctx context.Context, checkpointed bool) (Finalization, error) {
	dir, err := os.MkdirTemp("", "code-converge-finalize-")
	if err != nil {
		return Finalization{}, fmt.Errorf("create finalization workspace: %w", err)
	}
	defer os.RemoveAll(dir)
	schemaPath := filepath.Join(dir, "schema.json")
	messagePath := filepath.Join(dir, "message.json")
	if err := os.WriteFile(schemaPath, []byte(finalizationSchema), 0o600); err != nil {
		return Finalization{}, fmt.Errorf("write finalization schema: %w", err)
	}
	prompt := a.Config.FinalizePrompt
	if checkpointed {
		prompt += "\n\nSuccessful findings fixes were already committed as local checkpoints. Do not create an empty commit; publish the current branch, create a change request if needed, and verify applicable CI."
	}
	prompt += "\n\nReturn only the JSON object required by the supplied output schema. Report the actual outcomes of commit, push, change_request, and ci."
	args := append(modelArgs(a.Config.FinalizeModel, a.Config.FinalizeEffort), "exec", "--output-schema", schemaPath, "--output-last-message", messagePath, "-")
	if _, err := a.Runner.Run(ctx, runner.Invocation{Args: args, Stdin: prompt, Output: a.output()}); err != nil {
		return Finalization{}, err
	}
	message, err := os.ReadFile(messagePath)
	if err != nil {
		return Finalization{}, fmt.Errorf("read finalization response: %w", err)
	}
	return ParseFinalization(message)
}

func (a Adapter) output() func(runner.Output) {
	if a.Output == nil {
		return nil
	}
	return func(chunk runner.Output) { a.Output(chunk.Source, chunk.Data) }
}

func modelArgs(model, effort string) []string {
	args := []string{"-c", "model=" + strconv.Quote(model)}
	if effort != "" {
		args = append(args, "-c", "model_reasoning_effort="+strconv.Quote(effort))
	}
	return args
}

func parseStructuredReview(data []byte) (ReviewResult, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return ReviewResult{}, fmt.Errorf("parse structured review response: %w", err)
	}
	if err := validateStructuredReviewKeys(data); err != nil {
		return ReviewResult{}, fmt.Errorf("parse structured review response: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var response structuredReview
	if err := decoder.Decode(&response); err != nil {
		return ReviewResult{}, fmt.Errorf("parse structured review response: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ReviewResult{}, errors.New("structured review response contains trailing data")
	}
	if err := validateStructuredReview(response); err != nil {
		return ReviewResult{}, err
	}
	if len(*response.Findings) == 0 {
		return ReviewResult{Clean: true}, nil
	}
	var counts Counts
	for _, finding := range *response.Findings {
		switch *finding.Priority {
		case 0:
			counts.Critical++
		case 1:
			counts.High++
		case 2:
			counts.Medium++
		case 3:
			counts.Low++
		}
	}
	return ReviewResult{Counts: counts}, nil
}

func validateStructuredReviewKeys(data []byte) error {
	var response map[string]json.RawMessage
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}
	if err := requireExactJSONKeys(response, "findings", "overall_correctness", "overall_explanation", "overall_confidence_score"); err != nil {
		return fmt.Errorf("structured review response: %w", err)
	}
	var findings []json.RawMessage
	if err := json.Unmarshal(response["findings"], &findings); err != nil {
		return fmt.Errorf("structured review response findings: %w", err)
	}
	for _, rawFinding := range findings {
		var finding map[string]json.RawMessage
		if err := json.Unmarshal(rawFinding, &finding); err != nil {
			return fmt.Errorf("structured review response finding: %w", err)
		}
		if err := requireExactJSONKeys(finding, "title", "body", "confidence_score", "priority", "code_location"); err != nil {
			return fmt.Errorf("structured review response finding: %w", err)
		}
		var location map[string]json.RawMessage
		if err := json.Unmarshal(finding["code_location"], &location); err != nil {
			return fmt.Errorf("structured review response code location: %w", err)
		}
		if err := requireExactJSONKeys(location, "absolute_file_path", "line_range"); err != nil {
			return fmt.Errorf("structured review response code location: %w", err)
		}
		var lineRange map[string]json.RawMessage
		if err := json.Unmarshal(location["line_range"], &lineRange); err != nil {
			return fmt.Errorf("structured review response line range: %w", err)
		}
		if err := requireExactJSONKeys(lineRange, "start", "end"); err != nil {
			return fmt.Errorf("structured review response line range: %w", err)
		}
	}
	return nil
}

func requireExactJSONKeys(object map[string]json.RawMessage, names ...string) error {
	if len(object) != len(names) {
		return errors.New("contains unknown or missing fields")
	}
	for _, name := range names {
		if _, ok := object[name]; !ok {
			return errors.New("contains unknown or missing fields")
		}
	}
	return nil
}

func validateStructuredReview(response structuredReview) error {
	if response.Findings == nil || response.OverallCorrectness == nil || response.OverallExplanation == nil || response.OverallConfidenceScore == nil {
		return errors.New("structured review response is incomplete")
	}
	for _, finding := range *response.Findings {
		if finding.Title == nil || finding.Body == nil || finding.ConfidenceScore == nil || finding.Priority == nil || finding.CodeLocation == nil || finding.CodeLocation.AbsoluteFilePath == nil || finding.CodeLocation.LineRange == nil || finding.CodeLocation.LineRange.Start == nil || finding.CodeLocation.LineRange.End == nil {
			return errors.New("structured review response contains an incomplete finding")
		}
		if *finding.Priority < 0 || *finding.Priority > 3 {
			return errors.New("structured review response contains an invalid finding priority")
		}
	}
	return nil
}

func ParseFinalization(data []byte) (Finalization, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return Finalization{}, fmt.Errorf("parse finalization response: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var result Finalization
	if err := decoder.Decode(&result); err != nil {
		return Finalization{}, fmt.Errorf("parse finalization response: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Finalization{}, errors.New("finalization response contains trailing data")
	}
	if err := validateFinalization(result); err != nil {
		return Finalization{}, err
	}
	return result, nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("finalization response contains trailing data")
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delimiter {
	case '{':
		keys := make(map[string]struct{})
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return errors.New("JSON contains an invalid object key")
			}
			if _, exists := keys[name]; exists {
				return fmt.Errorf("JSON contains duplicate field %q", name)
			}
			keys[name] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("JSON contains an unclosed object")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("JSON contains an unclosed array")
		}
	default:
		return errors.New("JSON contains an unexpected delimiter")
	}
	return nil
}

func validateFinalization(result Finalization) error {
	validStep := func(value string) bool {
		return value == "success" || value == "skipped" || value == "failed" || value == "unknown"
	}
	if !validStep(result.Commit) || !validStep(result.Push) || !validStep(result.ChangeRequest) || !validStep(result.CI) {
		return errors.New("finalization response contains an invalid step status")
	}
	switch result.Verdict {
	case "SUCCESS":
		if !oneOf(result.Commit, "success", "skipped") || !oneOf(result.Push, "success", "skipped") || !oneOf(result.ChangeRequest, "success", "skipped") || !oneOf(result.CI, "success", "skipped") {
			return errors.New("SUCCESS is inconsistent with step outcomes")
		}
	case "CI_FAILED":
		if !oneOf(result.Commit, "success", "skipped") || !oneOf(result.Push, "success", "skipped") || !oneOf(result.ChangeRequest, "success", "skipped") || result.CI != "failed" {
			return errors.New("CI_FAILED is inconsistent with step outcomes")
		}
	case "FAILED":
		if oneOf(result.Commit, "success", "skipped") && oneOf(result.Push, "success", "skipped") && oneOf(result.ChangeRequest, "success", "skipped") && oneOf(result.CI, "success", "skipped") {
			return errors.New("FAILED is inconsistent with successful step outcomes")
		}
		if oneOf(result.Commit, "success", "skipped") && oneOf(result.Push, "success", "skipped") && oneOf(result.ChangeRequest, "success", "skipped") && result.CI == "failed" {
			return errors.New("FAILED is inconsistent with a CI-only failure")
		}
	default:
		return errors.New("finalization response contains an unknown verdict")
	}
	return nil
}

func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

const reviewSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["findings", "overall_correctness", "overall_explanation", "overall_confidence_score"],
  "properties": {
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "additionalProperties": false,
        "required": ["title", "body", "confidence_score", "priority", "code_location"],
        "properties": {
          "title": {"type": "string"},
          "body": {"type": "string"},
          "confidence_score": {"type": "number"},
          "priority": {"type": "integer", "enum": [0, 1, 2, 3]},
          "code_location": {
            "type": "object",
            "additionalProperties": false,
            "required": ["absolute_file_path", "line_range"],
            "properties": {
              "absolute_file_path": {"type": "string"},
              "line_range": {
                "type": "object",
                "additionalProperties": false,
                "required": ["start", "end"],
                "properties": {
                  "start": {"type": "integer"},
                  "end": {"type": "integer"}
                }
              }
            }
          }
        }
      }
    },
    "overall_correctness": {"type": "string"},
    "overall_explanation": {"type": "string"},
    "overall_confidence_score": {"type": "number"}
  }
}`

const finalizationSchema = `{
  "type": "object",
  "additionalProperties": false,
  "required": ["verdict", "commit", "push", "change_request", "ci"],
  "properties": {
    "verdict": {"type": "string", "enum": ["SUCCESS", "CI_FAILED", "FAILED"]},
    "commit": {"type": "string", "enum": ["success", "skipped", "failed", "unknown"]},
    "push": {"type": "string", "enum": ["success", "skipped", "failed", "unknown"]},
    "change_request": {"type": "string", "enum": ["success", "skipped", "failed", "unknown"]},
    "ci": {"type": "string", "enum": ["success", "skipped", "failed", "unknown"]}
  }
}`
