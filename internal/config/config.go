package config

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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
	LogFormat           OptionalString
	Heartbeat           OptionalString
	Color               OptionalString
	Mode                OptionalString
	MaxCycles           OptionalString
	MaxCIRecoveries     OptionalString
	CITimeout           OptionalString
	ReviewModel         OptionalString
	ReviewEffort        OptionalString
	FixModel            OptionalString
	FixEffort           OptionalString
	FixPromptPath       OptionalString
	CIFixModel          OptionalString
	CIFixEffort         OptionalString
	CIFixPromptPath     OptionalString
	ReviewBase          OptionalString
	SessionLogDir       OptionalString
	SessionLogRetention OptionalString
	NoSessionLog        bool
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

	LogFormat           string
	Heartbeat           time.Duration
	Color               string
	Mode                string
	MaxCycles           int
	MaxCIRecoveries     int
	CITimeout           time.Duration
	ReviewModel         string
	ReviewEffort        string
	FixModel            string
	FixEffort           string
	FixPrompt           string
	CIFixModel          string
	CIFixEffort         string
	CIFixPrompt         string
	ReviewBase          string
	SessionLogDir       string
	SessionLogRetention time.Duration
	NoSessionLog        bool

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

type fileConfig map[string]string

type yamlFileConfig struct {
	LogFormat           *string `yaml:"log-format"`
	Heartbeat           *string `yaml:"heartbeat"`
	Color               *string `yaml:"color"`
	Mode                *string `yaml:"mode"`
	MaxCycles           *int    `yaml:"max-cycles"`
	MaxCIRecoveries     *int    `yaml:"max-ci-recoveries"`
	CITimeout           *string `yaml:"ci-timeout"`
	ReviewModel         *string `yaml:"review-model"`
	ReviewEffort        *string `yaml:"review-reasoning-effort"`
	FixModel            *string `yaml:"fix-model"`
	FixEffort           *string `yaml:"fix-reasoning-effort"`
	FixPromptPath       *string `yaml:"fix-prompt-file"`
	CIFixModel          *string `yaml:"ci-fix-model"`
	CIFixEffort         *string `yaml:"ci-fix-reasoning-effort"`
	CIFixPromptPath     *string `yaml:"ci-fix-prompt-file"`
	ReviewBase          *string `yaml:"review-base"`
	SessionLogDir       *string `yaml:"session-log-dir"`
	SessionLogRetention *string `yaml:"session-log-retention"`
}

type stageProfile struct {
	reviewModel, reviewEffort string
	fixModel, fixEffort       string
	ciFixModel, ciFixEffort   string
}

func profileFor(mode string) (stageProfile, bool) {
	switch mode {
	case "fast":
		return stageProfile{
			reviewModel: "gpt-5.6-terra", reviewEffort: "medium",
			fixModel: "gpt-5.6-luna", fixEffort: "medium",
			ciFixModel: "gpt-5.6-luna", ciFixEffort: "medium",
		}, true
	case "best":
		return stageProfile{
			reviewModel: "gpt-5.6-sol", reviewEffort: "high",
			fixModel: "gpt-5.6-terra", fixEffort: "high",
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
	if err := rejectObsoleteFinalizeSettings(userDir, projectDir); err != nil {
		return Config{}, err
	}
	logFormat, logFormatSetting, err := resolve(spec{
		name: "log-format", file: "log-format", env: "CODE_CONVERGE_LOG_FORMAT", def: "human", builtIn: "human", defSource: SourceDefault, override: overrides.LogFormat,
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
		{name: "ci-timeout", file: "ci-timeout", env: "CODE_CONVERGE_CI_TIMEOUT", def: "60m", builtIn: "60m", defSource: SourceDefault, override: overrides.CITimeout},
		{name: "review-model", file: "review-model", env: "CODE_CONVERGE_REVIEW_MODEL", def: profile.reviewModel, builtIn: fast.reviewModel, defSource: profileSource, override: overrides.ReviewModel},
		{name: "review-reasoning-effort", file: "review-reasoning-effort", env: "CODE_CONVERGE_REVIEW_REASONING_EFFORT", def: profile.reviewEffort, builtIn: fast.reviewEffort, defSource: profileSource, override: overrides.ReviewEffort},
		{name: "fix-model", file: "fix-model", env: "CODE_CONVERGE_FIX_MODEL", def: profile.fixModel, builtIn: fast.fixModel, defSource: profileSource, override: overrides.FixModel},
		{name: "fix-reasoning-effort", file: "fix-reasoning-effort", env: "CODE_CONVERGE_FIX_REASONING_EFFORT", def: profile.fixEffort, builtIn: fast.fixEffort, defSource: profileSource, override: overrides.FixEffort},
		{name: "fix-prompt", file: "fix-prompt-file", env: "CODE_CONVERGE_FIX_PROMPT_FILE", def: "fix findings", builtIn: "fix findings", defSource: SourceDefault, override: overrides.FixPromptPath, promptFile: true},
		{name: "ci-fix-model", file: "ci-fix-model", env: "CODE_CONVERGE_CI_FIX_MODEL", def: profile.ciFixModel, builtIn: fast.ciFixModel, defSource: profileSource, override: overrides.CIFixModel},
		{name: "ci-fix-reasoning-effort", file: "ci-fix-reasoning-effort", env: "CODE_CONVERGE_CI_FIX_REASONING_EFFORT", def: profile.ciFixEffort, builtIn: fast.ciFixEffort, defSource: profileSource, override: overrides.CIFixEffort},
		{name: "ci-fix-prompt", file: "ci-fix-prompt-file", env: "CODE_CONVERGE_CI_FIX_PROMPT_FILE", def: "Исправь CI", builtIn: "Исправь CI", defSource: SourceDefault, override: overrides.CIFixPromptPath, promptFile: true},
		{name: "review-base", file: "review-base", env: "CODE_CONVERGE_REVIEW_BASE", def: "", builtIn: "", defSource: SourceDefault, override: overrides.ReviewBase},
		{name: "session-log-dir", file: "session-log-dir", env: "CODE_CONVERGE_SESSION_LOG_DIR", def: filepath.Join(home, ".code-converge", "session-logs"), builtIn: filepath.Join(home, ".code-converge", "session-logs"), defSource: SourceDefault, override: overrides.SessionLogDir},
		{name: "session-log-retention", file: "session-log-retention", env: "CODE_CONVERGE_SESSION_LOG_RETENTION", def: "24h", builtIn: "24h", defSource: SourceDefault, override: overrides.SessionLogRetention},
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
	ciTimeout, err := time.ParseDuration(strings.TrimSpace(values["ci-timeout"]))
	if err != nil || ciTimeout < time.Second {
		return Config{}, fmt.Errorf("ci-timeout must be a duration of at least 1s")
	}
	sessionLogDir, err := sessionLogPath(values["session-log-dir"], home)
	if err != nil {
		return Config{}, err
	}
	sessionLogRetention, err := sessionLogRetention(values["session-log-retention"])
	if err != nil {
		return Config{}, err
	}
	for index := range settings {
		switch settings[index].Name {
		case "session-log-dir":
			settings[index].Value = sessionLogDir
			settings[index].DisplayValue = sessionLogDir
			settings[index].Default = filepath.Join(home, ".code-converge", "session-logs")
			settings[index].DisplayDefault = settings[index].Default
		}
	}
	for _, name := range []string{"review-model", "review-reasoning-effort", "fix-model", "fix-reasoning-effort", "ci-fix-model", "ci-fix-reasoning-effort"} {
		if strings.TrimSpace(values[name]) == "" {
			return Config{}, fmt.Errorf("%s must not be empty", name)
		}
	}

	return Config{
		Root: root, LogFormat: logFormat, Heartbeat: heartbeat, Color: color,
		Mode: mode, MaxCycles: maxCycles, MaxCIRecoveries: maxCI, CITimeout: ciTimeout,
		ReviewModel: values["review-model"], ReviewEffort: values["review-reasoning-effort"],
		FixModel: values["fix-model"], FixEffort: values["fix-reasoning-effort"], FixPrompt: values["fix-prompt"],
		CIFixModel: values["ci-fix-model"], CIFixEffort: values["ci-fix-reasoning-effort"], CIFixPrompt: values["ci-fix-prompt"], Settings: settings,
		ReviewBase: values["review-base"], SessionLogDir: sessionLogDir, SessionLogRetention: sessionLogRetention, NoSessionLog: overrides.NoSessionLog,
	}, nil
}

// rejectObsoleteFinalizeSettings makes the deliberate Finalize-stage removal
// actionable.  Leaving a previously supported setting silently ignored would
// make an operator believe it still controls delivery behavior.
func rejectObsoleteFinalizeSettings(userDir, projectDir string) error {
	for _, name := range []string{
		"CODE_CONVERGE_FINALIZE_MODEL",
		"CODE_CONVERGE_FINALIZE_REASONING_EFFORT",
		"CODE_CONVERGE_FINALIZE_PROMPT_FILE",
	} {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			return fmt.Errorf("%s was removed; remove this obsolete Finalize-stage setting", name)
		}
	}
	return nil
}

func sessionLogPath(value, home string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("session-log-dir must not be empty")
	}
	if value == "~" {
		value = home
	} else if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, `~\`) {
		value = filepath.Join(home, value[2:])
	}
	if !filepath.IsAbs(value) {
		return "", fmt.Errorf("session-log-dir must be an absolute path")
	}
	return filepath.Clean(value), nil
}

func sessionLogRetention(value string) (time.Duration, error) {
	duration, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || duration < time.Second {
		return 0, fmt.Errorf("session-log-retention must be a duration of at least 1s")
	}
	return duration, nil
}

// ResolveLogFormat resolves only the presentation format so startup failures in
// unrelated settings can use the requested renderer. A format-resolution error
// intentionally leaves callers on the built-in human fallback.
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
		name: "log-format", file: "log-format", env: "CODE_CONVERGE_LOG_FORMAT", def: "human", builtIn: "human", defSource: SourceDefault, override: override,
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
		config, err := readFileConfig(candidate.dir)
		if err != nil {
			return "", Setting{}, err
		}
		content, ok := config[item.file]
		if !ok {
			continue
		}
		if item.promptFile {
			prompt, path, err := readConfiguredPrompt(candidate.dir, content)
			if err != nil {
				return "", Setting{}, fmt.Errorf("%s from %s config: %w", item.name, candidate.source, err)
			}
			value, display = prompt, path
		} else {
			value, display = strings.TrimSpace(content), strings.TrimSpace(content)
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

func readFileConfig(dir string) (fileConfig, error) {
	path := filepath.Join(dir, "config.yaml")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fileConfig{}, nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var decoded yamlFileConfig
	decoder := yaml.NewDecoder(strings.NewReader(string(content)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&decoded); err == io.EOF {
		return fileConfig{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("invalid YAML configuration %s: %w", path, err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("invalid YAML configuration %s: only one YAML document is permitted", path)
		}
		return nil, fmt.Errorf("invalid YAML configuration %s: %w", path, err)
	}
	values := fileConfig{}
	setString := func(key string, value *string) {
		if value != nil {
			values[key] = *value
		}
	}
	setInt := func(key string, value *int) {
		if value != nil {
			values[key] = strconv.Itoa(*value)
		}
	}
	setString("log-format", decoded.LogFormat)
	setString("heartbeat", decoded.Heartbeat)
	setString("color", decoded.Color)
	setString("mode", decoded.Mode)
	setInt("max-cycles", decoded.MaxCycles)
	setInt("max-ci-recoveries", decoded.MaxCIRecoveries)
	setString("ci-timeout", decoded.CITimeout)
	setString("review-model", decoded.ReviewModel)
	setString("review-reasoning-effort", decoded.ReviewEffort)
	setString("fix-model", decoded.FixModel)
	setString("fix-reasoning-effort", decoded.FixEffort)
	setString("fix-prompt-file", decoded.FixPromptPath)
	setString("ci-fix-model", decoded.CIFixModel)
	setString("ci-fix-reasoning-effort", decoded.CIFixEffort)
	setString("ci-fix-prompt-file", decoded.CIFixPromptPath)
	setString("review-base", decoded.ReviewBase)
	setString("session-log-dir", decoded.SessionLogDir)
	setString("session-log-retention", decoded.SessionLogRetention)
	return values, nil
}

func readConfiguredPrompt(configDir, value string) (string, string, error) {
	path := value
	if !filepath.IsAbs(path) {
		path = filepath.Join(configDir, path)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("read prompt file %s: %w", path, err)
	}
	return string(content), path, nil
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
