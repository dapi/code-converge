package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	SourceDefault = "built-in default"
	SourceEnv     = "environment"
	SourceUser    = "user"
	SourceProject = "project"
	SourceCLI     = "cli"
)

type OptionalString struct {
	Value string
	Set   bool
}

type Overrides struct {
	MaxCycles          OptionalString
	MaxCIRecoveries    OptionalString
	ReviewModel        OptionalString
	ReviewEffort       OptionalString
	FixModel           OptionalString
	FixEffort          OptionalString
	FixPromptPath      OptionalString
	FinalizeModel      OptionalString
	FinalizePromptPath OptionalString
	CIFixModel         OptionalString
	CIFixPromptPath    OptionalString
}

type Setting struct {
	Name           string
	Value          string
	Source         string
	Default        string
	DisplayValue   string
	DisplayDefault string
}

type Config struct {
	Root string

	MaxCycles       int
	MaxCIRecoveries int
	ReviewModel     string
	ReviewEffort    string
	FixModel        string
	FixEffort       string
	FixPrompt       string
	FinalizeModel   string
	FinalizePrompt  string
	CIFixModel      string
	CIFixPrompt     string

	Settings []Setting
}

type spec struct {
	name       string
	file       string
	env        string
	def        string
	override   OptionalString
	promptFile bool
}

func Load(cwd, home string, overrides Overrides) (Config, error) {
	root, err := FindGitRoot(cwd)
	if err != nil {
		return Config{}, err
	}
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("resolve user home: %w", err)
		}
	}
	projectDir := filepath.Join(root, ".reviewer")
	userDir := filepath.Join(home, ".reviewer")
	specs := []spec{
		{"max-cycles", "max-cycles", "REVIEWER_MAX_CYCLES", "10", overrides.MaxCycles, false},
		{"max-ci-recoveries", "max-ci-recoveries", "REVIEWER_MAX_CI_RECOVERIES", "3", overrides.MaxCIRecoveries, false},
		{"review-model", "review-model", "REVIEWER_REVIEW_MODEL", "gpt-5.6-sol", overrides.ReviewModel, false},
		{"review-reasoning-effort", "review-reasoning-effort", "REVIEWER_REVIEW_REASONING_EFFORT", "medium", overrides.ReviewEffort, false},
		{"fix-model", "fix-model", "REVIEWER_FIX_MODEL", "gpt-5.6-luna", overrides.FixModel, false},
		{"fix-reasoning-effort", "fix-reasoning-effort", "REVIEWER_FIX_REASONING_EFFORT", "medium", overrides.FixEffort, false},
		{"fix-prompt", "fix-findings.md", "REVIEWER_FIX_PROMPT_FILE", "fix findings", overrides.FixPromptPath, true},
		{"finalize-model", "finalize-model", "REVIEWER_FINALIZE_MODEL", "gpt-5.3-codex-spark", overrides.FinalizeModel, false},
		{"finalize-prompt", "finalize.md", "REVIEWER_FINALIZE_PROMPT_FILE", "commit, push, create PR, ensure CI is green", overrides.FinalizePromptPath, true},
		{"ci-fix-model", "ci-fix-model", "REVIEWER_CI_FIX_MODEL", "", overrides.CIFixModel, false},
		{"ci-fix-prompt", "fix-ci.md", "REVIEWER_CI_FIX_PROMPT_FILE", "Исправь CI", overrides.CIFixPromptPath, true},
	}

	values := make(map[string]string, len(specs))
	settings := make([]Setting, 0, len(specs))
	for _, item := range specs {
		value, setting, resolveErr := resolve(item, cwd, userDir, projectDir)
		if resolveErr != nil {
			return Config{}, resolveErr
		}
		values[item.name] = value
		settings = append(settings, setting)
	}

	maxCycles, err := nonNegative("max-cycles", values["max-cycles"])
	if err != nil {
		return Config{}, err
	}
	maxCI, err := nonNegative("max-ci-recoveries", values["max-ci-recoveries"])
	if err != nil {
		return Config{}, err
	}
	for _, name := range []string{"review-model", "review-reasoning-effort", "fix-model", "fix-reasoning-effort", "finalize-model"} {
		if strings.TrimSpace(values[name]) == "" {
			return Config{}, fmt.Errorf("%s must not be empty", name)
		}
	}

	return Config{
		Root: root, MaxCycles: maxCycles, MaxCIRecoveries: maxCI,
		ReviewModel: values["review-model"], ReviewEffort: values["review-reasoning-effort"],
		FixModel: values["fix-model"], FixEffort: values["fix-reasoning-effort"], FixPrompt: values["fix-prompt"],
		FinalizeModel: values["finalize-model"], FinalizePrompt: values["finalize-prompt"],
		CIFixModel: values["ci-fix-model"], CIFixPrompt: values["ci-fix-prompt"], Settings: settings,
	}, nil
}

func resolve(item spec, cwd, userDir, projectDir string) (string, Setting, error) {
	value, source := item.def, SourceDefault
	display := displayDefault(item)
	if envValue, ok := os.LookupEnv(item.env); ok {
		if item.promptFile {
			content, path, err := readExplicitPrompt(cwd, envValue)
			if err != nil {
				return "", Setting{}, fmt.Errorf("%s from environment: %w", item.name, err)
			}
			value, display = content, path
		} else {
			value, display = strings.TrimSpace(envValue), strings.TrimSpace(envValue)
		}
		source = SourceEnv
	}
	for _, candidate := range []struct{ dir, source string }{{userDir, SourceUser}, {projectDir, SourceProject}} {
		path := filepath.Join(candidate.dir, item.file)
		content, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", Setting{}, fmt.Errorf("read %s: %w", path, err)
		}
		if item.promptFile {
			value, display = string(content), path
		} else {
			value, display = strings.TrimSpace(string(content)), strings.TrimSpace(string(content))
		}
		source = candidate.source
	}
	if item.override.Set {
		if item.promptFile {
			content, path, err := readExplicitPrompt(cwd, item.override.Value)
			if err != nil {
				return "", Setting{}, fmt.Errorf("%s from cli: %w", item.name, err)
			}
			value, display = content, path
		} else {
			value, display = strings.TrimSpace(item.override.Value), strings.TrimSpace(item.override.Value)
		}
		source = SourceCLI
	}
	return value, Setting{Name: item.name, Value: value, Source: source, Default: item.def, DisplayValue: display, DisplayDefault: displayDefault(item)}, nil
}

func displayDefault(item spec) string {
	if item.promptFile {
		return strconv.Quote(item.def)
	}
	if item.def == "" {
		return "agent-default"
	}
	return item.def
}

func readExplicitPrompt(cwd, path string) (string, string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(cwd, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read prompt file %s: %w", path, err)
	}
	return string(content), path, nil
}

func nonNegative(name, value string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return n, nil
}

func FindGitRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve current directory: %w", err)
	}
	command := exec.Command("git", "-C", current, "rev-parse", "--show-toplevel")
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("target directory is not inside a Git repository")
	}
	root := strings.TrimSpace(string(output))
	if root == "" {
		return "", fmt.Errorf("Git returned an empty repository root")
	}
	return filepath.Clean(root), nil
}

func Format(cfg Config) string {
	var out strings.Builder
	for _, setting := range cfg.Settings {
		fmt.Fprintf(&out, "%s: %s (%s", setting.Name, setting.DisplayValue, setting.Source)
		if setting.Source != SourceDefault {
			fmt.Fprintf(&out, "; built-in: %s", setting.DisplayDefault)
		}
		out.WriteString(")\n")
	}
	return out.String()
}
