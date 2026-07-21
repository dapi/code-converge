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
	"regexp"
	"strconv"
	"strings"

	"github.com/dapi/reviewer/internal/config"
	"github.com/dapi/reviewer/internal/runner"
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
}

type Finalization struct {
	Verdict       string `json:"verdict"`
	Commit        string `json:"commit"`
	Push          string `json:"push"`
	ChangeRequest string `json:"change_request"`
	CI            string `json:"ci"`
}

type Adapter struct {
	Runner runner.Runner
	Config config.Config
}

func (a Adapter) Review(ctx context.Context) (ReviewResult, error) {
	result, err := a.Runner.Run(ctx, runner.Invocation{Args: append(modelArgs(a.Config.ReviewModel, a.Config.ReviewEffort), "review", "--uncommitted")})
	if err != nil {
		return ReviewResult{}, err
	}
	review, err := ParseReview(result.Stdout)
	if err != nil {
		return ReviewResult{}, err
	}
	review.Report = strings.TrimSpace(ansiPattern.ReplaceAllString(result.Stdout, ""))
	return review, nil
}

func (a Adapter) FixFindings(ctx context.Context, report string) error {
	prompt := a.Config.FixPrompt + "\n\nReview findings to address:\n\n" + report
	_, err := a.Runner.Run(ctx, runner.Invocation{Args: append(modelArgs(a.Config.FixModel, a.Config.FixEffort), "exec", "-"), Stdin: prompt})
	return err
}

func (a Adapter) FixCI(ctx context.Context) error {
	args := []string{}
	if strings.TrimSpace(a.Config.CIFixModel) != "" {
		args = append(args, "-c", "model="+strconv.Quote(a.Config.CIFixModel))
	}
	args = append(args, "exec", "-")
	_, err := a.Runner.Run(ctx, runner.Invocation{Args: args, Stdin: a.Config.CIFixPrompt})
	return err
}

func (a Adapter) Finalize(ctx context.Context) (Finalization, error) {
	dir, err := os.MkdirTemp("", "reviewer-finalize-")
	if err != nil {
		return Finalization{}, fmt.Errorf("create finalization workspace: %w", err)
	}
	defer os.RemoveAll(dir)
	schemaPath := filepath.Join(dir, "schema.json")
	messagePath := filepath.Join(dir, "message.json")
	if err := os.WriteFile(schemaPath, []byte(finalizationSchema), 0o600); err != nil {
		return Finalization{}, fmt.Errorf("write finalization schema: %w", err)
	}
	prompt := a.Config.FinalizePrompt + "\n\nReturn only the JSON object required by the supplied output schema. Report the actual outcomes of commit, push, change_request, and ci."
	args := append(modelArgs(a.Config.FinalizeModel, ""), "exec", "--output-schema", schemaPath, "--output-last-message", messagePath, "-")
	if _, err := a.Runner.Run(ctx, runner.Invocation{Args: args, Stdin: prompt}); err != nil {
		return Finalization{}, err
	}
	message, err := os.ReadFile(messagePath)
	if err != nil {
		return Finalization{}, fmt.Errorf("read finalization response: %w", err)
	}
	return ParseFinalization(message)
}

func modelArgs(model, effort string) []string {
	args := []string{"-c", "model=" + strconv.Quote(model)}
	if effort != "" {
		args = append(args, "-c", "model_reasoning_effort="+strconv.Quote(effort))
	}
	return args
}

var (
	ansiPattern          = regexp.MustCompile(`\x1b\[[0-9;]*[[:alpha:]]`)
	bracketedFindingLine = regexp.MustCompile(`^\s*(?:#{1,6}\s+|[-*+]\s+|\d+[.)]\s+)?\[([^]]+)\]\s+.+$`)
	priorityToken        = regexp.MustCompile(`^P[0-9]+$`)
	findingHeading       = regexp.MustCompile(`(?i)^#{1,6}\s+findings\s*:?[[:space:]]*$`)
)

func ParseReview(raw string) (ReviewResult, error) {
	text := strings.TrimSpace(ansiPattern.ReplaceAllString(raw, ""))
	if text == "" {
		return ReviewResult{}, errors.New("empty review report")
	}
	var counts Counts
	inFindings := false
	found := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if findingHeading.MatchString(trimmed) {
			inFindings = true
			continue
		}
		if strings.HasPrefix(trimmed, "#") && inFindings {
			inFindings = false
		}
		match := bracketedFindingLine.FindStringSubmatch(line)
		if len(match) == 2 {
			priority := strings.ToUpper(match[1])
			if !priorityToken.MatchString(priority) {
				return ReviewResult{}, fmt.Errorf("review report contains unsupported finding label %q", match[1])
			}
			addPriority(&counts, priority)
			found++
			continue
		}
	}
	if found > 0 {
		if containsExplicitCleanLine(text) {
			return ReviewResult{}, errors.New("review report mixes findings with a clean verdict")
		}
		return ReviewResult{Counts: counts}, nil
	}
	if isExplicitCleanReport(text) {
		return ReviewResult{Clean: true}, nil
	}
	return ReviewResult{}, errors.New("review report is not safely classifiable")
}

func containsExplicitCleanLine(text string) bool {
	for _, line := range strings.Split(text, "\n") {
		if isExplicitClean(line) {
			return true
		}
	}
	return false
}

func addPriority(counts *Counts, priority string) {
	switch priority {
	case "P0":
		counts.Critical++
	case "P1":
		counts.High++
	case "P2":
		counts.Medium++
	case "P3":
		counts.Low++
	default:
		counts.Unknown++
	}
}

func isExplicitClean(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "- "))
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "* "))
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "## review"))
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "## findings"))
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, ":"))
	allowed := map[string]bool{
		"none found": true, "none found.": true,
		"no findings": true, "no findings.": true,
		"no findings found": true, "no findings found.": true,
		"no issues found": true, "no issues found.": true,
		"no actionable findings": true, "no actionable findings.": true,
		"the review found no issues": true, "the review found no issues.": true,
		"the review found no findings": true, "the review found no findings.": true,
		"i found no issues in the changes": true, "i found no issues in the changes.": true,
	}
	return allowed[normalized]
}

// isExplicitCleanReport accepts only a heading plus an allowlisted clean line
// (or a standalone allowlisted clean line). Arbitrary prose is rejected.
func isExplicitCleanReport(text string) bool {
	cleanLine := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || findingHeading.MatchString(trimmed) || strings.EqualFold(trimmed, "## review") {
			continue
		}
		if isExplicitClean(trimmed) {
			if cleanLine {
				return false
			}
			cleanLine = true
			continue
		}
		return false
	}
	return cleanLine
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
				return errors.New("finalization response contains an invalid object key")
			}
			if _, exists := keys[name]; exists {
				return fmt.Errorf("finalization response contains duplicate field %q", name)
			}
			keys[name] = struct{}{}
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("finalization response contains an unclosed object")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("finalization response contains an unclosed array")
		}
	default:
		return errors.New("finalization response contains an unexpected delimiter")
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
