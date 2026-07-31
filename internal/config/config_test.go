package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var codeConvergeEnv = []string{
	"CODE_CONVERGE_LOG_FORMAT", "CODE_CONVERGE_HEARTBEAT", "CODE_CONVERGE_COLOR", "CODE_CONVERGE_MODE",
	"CODE_CONVERGE_MAX_CYCLES", "CODE_CONVERGE_MAX_CI_RECOVERIES", "CODE_CONVERGE_CI_TIMEOUT", "CODE_CONVERGE_REVIEW_MODEL", "CODE_CONVERGE_REVIEW_REASONING_EFFORT",
	"CODE_CONVERGE_FIX_MODEL", "CODE_CONVERGE_FIX_REASONING_EFFORT", "CODE_CONVERGE_FIX_PROMPT_FILE", "CODE_CONVERGE_FINALIZE_MODEL",
	"CODE_CONVERGE_FINALIZE_REASONING_EFFORT", "CODE_CONVERGE_FINALIZE_PROMPT_FILE", "CODE_CONVERGE_CI_FIX_MODEL",
	"CODE_CONVERGE_CI_FIX_REASONING_EFFORT", "CODE_CONVERGE_CI_FIX_PROMPT_FILE", "CODE_CONVERGE_REVIEW_BASE",
	"CODE_CONVERGE_SESSION_LOG_DIR", "CODE_CONVERGE_SESSION_LOG_RETENTION",
}

func TestMain(m *testing.M) {
	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE"} {
		_ = os.Unsetenv(name)
	}
	os.Exit(m.Run())
}

func TestYAMLPrecedenceAndConfigReport(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	t.Setenv("CODE_CONVERGE_MAX_CYCLES", "2")
	writeConfig(t, home, "max-cycles: 3\nmode: best\n")
	writeConfig(t, root, "max-cycles: 4\nmode: fast\n")
	cfg, err := Load(root, home, Overrides{MaxCycles: OptionalString{Value: "5", Set: true}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxCycles != 5 || cfg.Mode != "fast" || source(cfg, "max-cycles") != SourceCLI || source(cfg, "mode") != SourceProject {
		t.Fatalf("cfg = %#v", cfg)
	}
	if got := Format(cfg); !strings.Contains(got, "max-cycles: 5 (cli; built-in: 10)") || !strings.Contains(got, "mode: fast (project)") {
		t.Fatalf("config output:\n%s", got)
	}
	cfg, err = Load(root, home, Overrides{})
	if err != nil || cfg.MaxCycles != 4 || source(cfg, "max-cycles") != SourceProject {
		t.Fatalf("project precedence: %#v, %v", cfg, err)
	}
	if err := os.Remove(filepath.Join(root, ".code-converge", "config.yaml")); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(root, home, Overrides{})
	if err != nil || cfg.MaxCycles != 3 || source(cfg, "max-cycles") != SourceUser {
		t.Fatalf("user precedence: %#v, %v", cfg, err)
	}
	if err := os.Remove(filepath.Join(home, ".code-converge", "config.yaml")); err != nil {
		t.Fatal(err)
	}
	cfg, err = Load(root, home, Overrides{})
	if err != nil || cfg.MaxCycles != 2 || source(cfg, "max-cycles") != SourceEnv {
		t.Fatalf("environment precedence: %#v, %v", cfg, err)
	}
}

func TestYAMLRepresentsAllSettings(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	write(t, filepath.Join(root, ".code-converge", "prompts", "fix.md"), "fix prompt\n")
	write(t, filepath.Join(root, ".code-converge", "prompts", "ci.md"), "ci prompt\n")
	writeConfig(t, root, strings.Join([]string{
		"log-format: human", "heartbeat: 2s", "color: never", "mode: best", "max-cycles: 1", "max-ci-recoveries: 2", "ci-timeout: 30m",
		"review-model: review", "review-reasoning-effort: low", "fix-model: fix", "fix-reasoning-effort: medium", "fix-prompt-file: prompts/fix.md",
		"ci-fix-model: ci", "ci-fix-reasoning-effort: medium", "ci-fix-prompt-file: prompts/ci.md",
		"review-base: main", "session-log-dir: " + filepath.Join(home, "logs"), "session-log-retention: 2h",
	}, "\n")+"\n")
	cfg, err := Load(root, home, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Heartbeat != 2*time.Second || cfg.MaxCycles != 1 || cfg.MaxCIRecoveries != 2 || cfg.CITimeout != 30*time.Minute || cfg.ReviewModel != "review" || cfg.FixPrompt != "fix prompt\n" || cfg.CIFixPrompt != "ci prompt\n" || cfg.ReviewBase != "main" || cfg.SessionLogRetention != 2*time.Hour {
		t.Fatalf("cfg = %#v", cfg)
	}
	for _, name := range []string{
		"log-format", "heartbeat", "color", "mode", "max-cycles", "max-ci-recoveries", "ci-timeout",
		"review-model", "review-reasoning-effort", "fix-model", "fix-reasoning-effort", "fix-prompt",
		"ci-fix-model", "ci-fix-reasoning-effort", "ci-fix-prompt",
		"review-base", "session-log-dir", "session-log-retention",
	} {
		if source(cfg, name) != SourceProject {
			t.Errorf("%s source = %q", name, source(cfg, name))
		}
	}
}

func TestYAMLValidation(t *testing.T) {
	for _, test := range []struct{ name, contents, want string }{
		{"malformed", "mode: [fast\n", "did not find expected"},
		{"unknown key", "typo: true\n", "field typo not found"},
		{"nested value", "mode:\n  value: fast\n", "cannot unmarshal"},
		{"duplicate", "mode: fast\nmode: best\n", "mapping key \"mode\" already defined"},
		{"invalid mode", "mode: invalid\n", "mode must be one of"},
		{"invalid integer", "max-cycles: -1\n", "max-cycles must be a non-negative integer"},
		{"invalid duration", "session-log-retention: 0\n", "session-log-retention must be a duration of at least 1s"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cleanEnv(t)
			root, home := repo(t)
			writeConfig(t, root, test.contents)
			_, err := Load(root, home, Overrides{})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestYAMLPromptPathFailureIsActionable(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	writeConfig(t, root, "fix-prompt-file: prompts/missing.md\n")
	_, err := Load(root, home, Overrides{})
	if err == nil || !strings.Contains(err.Error(), "fix-prompt from project config") || !strings.Contains(err.Error(), "prompts/missing.md") {
		t.Fatalf("error = %v", err)
	}
}

func TestLegacyFilesAreIgnored(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	write(t, filepath.Join(home, ".code-converge", "mode"), "best\n")
	write(t, filepath.Join(root, ".code-converge", "max-cycles"), "1\n")
	write(t, filepath.Join(root, ".code-converge", "fix-findings.md"), "legacy prompt\n")
	cfg, err := Load(root, home, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != "fast" || cfg.MaxCycles != 10 || cfg.FixPrompt != "fix findings" || source(cfg, "mode") != SourceDefault {
		t.Fatalf("legacy file was read: %#v", cfg)
	}
}

func TestCLIAndEnvironmentStillWin(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	t.Setenv("CODE_CONVERGE_COLOR", "never")
	writeConfig(t, home, "color: auto\n")
	writeConfig(t, root, "color: never\n")
	cfg, err := Load(root, home, Overrides{Color: OptionalString{Value: "auto", Set: true}})
	if err != nil || cfg.Color != "auto" || source(cfg, "color") != SourceCLI {
		t.Fatalf("cfg=%#v err=%v", cfg, err)
	}
}

func TestResolveLogFormatReadsYAML(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	writeConfig(t, root, "log-format: kv\n")
	format, err := ResolveLogFormat(root, home, OptionalString{})
	if err != nil || format != "kv" {
		t.Fatalf("format=%q err=%v", format, err)
	}
}

func TestDefaultsAndProfileResolution(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	writeConfig(t, root, "mode: best\n")
	cfg, err := Load(root, home, Overrides{})
	if err != nil || cfg.ReviewModel != "gpt-5.6-sol" || cfg.FixModel != "gpt-5.6-terra" || source(cfg, "review-model") != "best profile" {
		t.Fatalf("cfg=%#v err=%v", cfg, err)
	}
}

func cleanEnv(t *testing.T) {
	t.Helper()
	for _, name := range codeConvergeEnv {
		_ = os.Unsetenv(name)
	}
}
func repo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	return root, t.TempDir()
}
func write(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}
func writeConfig(t *testing.T, directory, value string) {
	t.Helper()
	write(t, filepath.Join(directory, ".code-converge", "config.yaml"), value)
}
func source(cfg Config, name string) string {
	for _, setting := range cfg.Settings {
		if setting.Name == name {
			return setting.Source
		}
	}
	return ""
}
