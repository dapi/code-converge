package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	LogFormat          OptionalString
	Heartbeat          OptionalString
	Color              OptionalString
	Mode               OptionalString
	MaxCycles          OptionalString
	MaxCIRecoveries    OptionalString
	ReviewModel        OptionalString
	ReviewEffort       OptionalString
	FixModel           OptionalString
	FixEffort          OptionalString
	FixPromptPath      OptionalString
	FinalizeModel      OptionalString
	FinalizeEffort     OptionalString
	FinalizePromptPath OptionalString
	CIFixModel         OptionalString
	CIFixEffort        OptionalString
	CIFixPromptPath    OptionalString
	ReviewBase         OptionalString
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

	LogFormat       string
	Heartbeat       time.Duration
	Color           string
	Mode            string
	MaxCycles       int
	MaxCIRecoveries int
	ReviewModel     string
	ReviewEffort    string
	FixModel        string
	FixEffort       string
	FixPrompt       string
	FinalizeModel   string
	FinalizeEffort  string
	FinalizePrompt  string
	CIFixModel      string
	CIFixEffort     string
	CIFixPrompt     string
	ReviewBase      string

	Settings []Setting
}

type spec struct {
	name       string
	file       string
	env        string
	def        string
	builtIn    string
	defSource  string
	override   OptionalString
	promptFile bool
}

type stageProfile struct {
	reviewModel, reviewEffort     string
	fixModel, fixEffort           string
	finalizeModel, finalizeEffort string
	ciFixModel, ciFixEffort       string
}

func profileFor(mode string) (stageProfile, bool) {
	switch mode {
	case "fast":
		return stageProfile{
			reviewModel: "gpt-5.6-terra", reviewEffort: "medium",
			fixModel: "gpt-5.6-luna", fixEffort: "medium",
			finalizeModel: "gpt-5.6-luna", finalizeEffort: "medium",
			ciFixModel: "gpt-5.6-luna", ciFixEffort: "medium",
		}, true
	case "best":
		return stageProfile{
			reviewModel: "gpt-5.6-sol", reviewEffort: "high",
			fixModel: "gpt-5.6-terra", fixEffort: "high",
			finalizeModel: "gpt-5.6-luna", finalizeEffort: "medium",
			ciFixModel: "gpt-5.6-terra", ciFixEffort: "high",
		}, true
	default:
		return stageProfile{}, false
	}
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
	projectDir := filepath.Join(root, ".code-converge")
	userDir := filepath.Join(home, ".code-converge")
	logFormat, logFormatSetting, err := resolve(spec{
		name: "log-format", file: "log-format", env: "CODE_CONVERGE_LOG_FORMAT", def: "kv", builtIn: "kv", defSource: SourceDefault, override: overrides.LogFormat,
	}, cwd, userDir, projectDir)
	if err != nil {
		return Config{}, err
	}
	if logFormat != "kv" && logFormat != "human" {
		return Config{}, fmt.Errorf("log-format must be one of: kv, human")
	}
	color, colorSetting, err := resolve(spec{
		name: "color", file: "color", env: "CODE_CONVERGE_COLOR", def: "auto", builtIn: "auto", defSource: SourceDefault, override: overrides.Color,
	}, cwd, userDir, projectDir)
	if err != nil {
		return Config{}, err
	}
	if color != "auto" && color != "never" {
		return Config{}, fmt.Errorf("color must be one of: auto, never")
	}
	heartbeatValue, heartbeatSetting, err := resolve(spec{
		name: "heartbeat", file: "heartbeat", env: "CODE_CONVERGE_HEARTBEAT", def: "0", builtIn: "0", defSource: SourceDefault, override: overrides.Heartbeat,
	}, cwd, userDir, projectDir)
	if err != nil {
		return Config{}, err
	}
	heartbeat, err := parseHeartbeat(heartbeatValue)
	if err != nil {
		return Config{}, err
	}
	if logFormat == "kv" && heartbeat > 0 {
		return Config{}, fmt.Errorf("heartbeat requires log-format=human")
	}
	mode, modeSetting, err := resolve(spec{
		name: "mode", file: "mode", env: "CODE_CONVERGE_MODE", def: "fast", builtIn: "fast", defSource: SourceDefault, override: overrides.Mode,
	}, cwd, userDir, projectDir)
	if err != nil {
		return Config{}, err
	}
	profile, ok := profileFor(mode)
	if !ok {
		return Config{}, fmt.Errorf("mode must be one of: fast, best")
	}
	fast, _ := profileFor("fast")
	profileSource := mode + " profile"
	specs := []spec{
		{name: "max-cycles", file: "max-cycles", env: "CODE_CONVERGE_MAX_CYCLES", def: "10", builtIn: "10", defSource: SourceDefault, override: overrides.MaxCycles},
		{name: "max-ci-recoveries", file: "max-ci-recoveries", env: "CODE_CONVERGE_MAX_CI_RECOVERIES", def: "3", builtIn: "3", defSource: SourceDefault, override: overrides.MaxCIRecoveries},
		{name: "review-model", file: "review-model", env: "CODE_CONVERGE_REVIEW_MODEL", def: profile.reviewModel, builtIn: fast.reviewModel, defSource: profileSource, override: overrides.ReviewModel},
		{name: "review-reasoning-effort", file: "review-reasoning-effort", env: "CODE_CONVERGE_REVIEW_REASONING_EFFORT", def: profile.reviewEffort, builtIn: fast.reviewEffort, defSource: profileSource, override: overrides.ReviewEffort},
		{name: "fix-model", file: "fix-model", env: "CODE_CONVERGE_FIX_MODEL", def: profile.fixModel, builtIn: fast.fixModel, defSource: profileSource, override: overrides.FixModel},
		{name: "fix-reasoning-effort", file: "fix-reasoning-effort", env: "CODE_CONVERGE_FIX_REASONING_EFFORT", def: profile.fixEffort, builtIn: fast.fixEffort, defSource: profileSource, override: overrides.FixEffort},
		{name: "fix-prompt", file: "fix-findings.md", env: "CODE_CONVERGE_FIX_PROMPT_FILE", def: "fix findings", builtIn: "fix findings", defSource: SourceDefault, override: overrides.FixPromptPath, promptFile: true},
		{name: "finalize-model", file: "finalize-model", env: "CODE_CONVERGE_FINALIZE_MODEL", def: profile.finalizeModel, builtIn: fast.finalizeModel, defSource: profileSource, override: overrides.FinalizeModel},
		{name: "finalize-reasoning-effort", file: "finalize-reasoning-effort", env: "CODE_CONVERGE_FINALIZE_REASONING_EFFORT", def: profile.finalizeEffort, builtIn: fast.finalizeEffort, defSource: profileSource, override: overrides.FinalizeEffort},
		{name: "finalize-prompt", file: "finalize.md", env: "CODE_CONVERGE_FINALIZE_PROMPT_FILE", def: "commit, push, create PR, ensure CI is green", builtIn: "commit, push, create PR, ensure CI is green", defSource: SourceDefault, override: overrides.FinalizePromptPath, promptFile: true},
		{name: "ci-fix-model", file: "ci-fix-model", env: "CODE_CONVERGE_CI_FIX_MODEL", def: profile.ciFixModel, builtIn: fast.ciFixModel, defSource: profileSource, override: overrides.CIFixModel},
		{name: "ci-fix-reasoning-effort", file: "ci-fix-reasoning-effort", env: "CODE_CONVERGE_CI_FIX_REASONING_EFFORT", def: profile.ciFixEffort, builtIn: fast.ciFixEffort, defSource: profileSource, override: overrides.CIFixEffort},
		{name: "ci-fix-prompt", file: "fix-ci.md", env: "CODE_CONVERGE_CI_FIX_PROMPT_FILE", def: "Исправь CI", builtIn: "Исправь CI", defSource: SourceDefault, override: overrides.CIFixPromptPath, promptFile: true},
		{name: "review-base", file: "review-base", env: "CODE_CONVERGE_REVIEW_BASE", def: "", builtIn: "", defSource: SourceDefault, override: overrides.ReviewBase},
	}

	values := make(map[string]string, len(specs))
	settings := make([]Setting, 0, len(specs)+4)
	settings = append(settings, logFormatSetting, heartbeatSetting, colorSetting, modeSetting)
	for _, item := range specs {
		value, setting, resolveErr := resolve(item, cwd, userDir, projectDir)
		if resolveErr != nil {
			return Config{}, resolveErr
		}
		values[item.name] = value
		if item.name == "review-base" && value == "" {
			setting.DisplayValue = "discover"
			setting.DisplayDefault = "discover"
		}
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
	for _, name := range []string{"review-model", "review-reasoning-effort", "fix-model", "fix-reasoning-effort", "finalize-model", "finalize-reasoning-effort", "ci-fix-model", "ci-fix-reasoning-effort"} {
		if strings.TrimSpace(values[name]) == "" {
			return Config{}, fmt.Errorf("%s must not be empty", name)
		}
	}

	return Config{
		Root: root, LogFormat: logFormat, Heartbeat: heartbeat, Color: color,
		Mode: mode, MaxCycles: maxCycles, MaxCIRecoveries: maxCI,
		ReviewModel: values["review-model"], ReviewEffort: values["review-reasoning-effort"],
		FixModel: values["fix-model"], FixEffort: values["fix-reasoning-effort"], FixPrompt: values["fix-prompt"],
		FinalizeModel: values["finalize-model"], FinalizeEffort: values["finalize-reasoning-effort"], FinalizePrompt: values["finalize-prompt"],
		CIFixModel: values["ci-fix-model"], CIFixEffort: values["ci-fix-reasoning-effort"], CIFixPrompt: values["ci-fix-prompt"], Settings: settings,
		ReviewBase: values["review-base"],
	}, nil
}

// ResolveLogFormat resolves only the presentation format so startup failures in
// unrelated settings can use the requested renderer. A format-resolution error
// intentionally leaves callers on the legacy kv fallback.
func ResolveLogFormat(cwd, home string, override OptionalString) (string, error) {
	root, err := FindGitRoot(cwd)
	if err != nil {
		return "", err
	}
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home: %w", err)
		}
	}
	value, _, err := resolve(spec{
		name: "log-format", file: "log-format", env: "CODE_CONVERGE_LOG_FORMAT", def: "kv", builtIn: "kv", defSource: SourceDefault, override: override,
	}, cwd, filepath.Join(home, ".code-converge"), filepath.Join(root, ".code-converge"))
	if err != nil {
		return "", err
	}
	if value != "kv" && value != "human" {
		return "", fmt.Errorf("log-format must be one of: kv, human")
	}
	return value, nil
}

func parseHeartbeat(value string) (time.Duration, error) {
	if strings.TrimSpace(value) == "0" {
		return 0, nil
	}
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || duration < time.Second {
		return 0, fmt.Errorf("heartbeat must be 0 or a duration of at least 1s")
	}
	return duration, nil
}

func resolve(item spec, cwd, userDir, projectDir string) (string, Setting, error) {
	value, source := item.def, item.defSource
	display := displayValue(item.def, item.promptFile)
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
	return value, Setting{Name: item.name, Value: value, Source: source, Default: item.builtIn, DisplayValue: display, DisplayDefault: displayDefault(item)}, nil
}

func displayDefault(item spec) string {
	return displayValue(item.builtIn, item.promptFile)
}

func displayValue(value string, promptFile bool) string {
	if promptFile {
		return strconv.Quote(value)
	}
	if value == "" {
		return "agent-default"
	}
	return value
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
		if setting.Value != setting.Default {
			fmt.Fprintf(&out, "; built-in: %s", setting.DisplayDefault)
		}
		out.WriteString(")\n")
	}
	return out.String()
}
