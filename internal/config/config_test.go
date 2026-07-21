package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var reviewerEnv = []string{
	"REVIEWER_MAX_CYCLES", "REVIEWER_MAX_CI_RECOVERIES", "REVIEWER_REVIEW_MODEL", "REVIEWER_REVIEW_REASONING_EFFORT",
	"REVIEWER_FIX_MODEL", "REVIEWER_FIX_REASONING_EFFORT", "REVIEWER_FIX_PROMPT_FILE", "REVIEWER_FINALIZE_MODEL",
	"REVIEWER_FINALIZE_PROMPT_FILE", "REVIEWER_CI_FIX_MODEL", "REVIEWER_CI_FIX_PROMPT_FILE",
}

func cleanEnv(t *testing.T) {
	t.Helper()
	for _, name := range reviewerEnv {
		t.Setenv(name, "")
		_ = os.Unsetenv(name)
	}
}

func repo(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	if output, err := exec.Command("git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	home := t.TempDir()
	return root, home
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

func TestLoadPrecedence(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	t.Setenv("REVIEWER_MAX_CYCLES", "2")
	write(t, filepath.Join(home, ".reviewer", "max-cycles"), "3\n")
	write(t, filepath.Join(root, ".reviewer", "max-cycles"), "4\n")
	cfg, err := Load(root, home, Overrides{MaxCycles: OptionalString{Value: "5", Set: true}})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxCycles != 5 || source(cfg, "max-cycles") != SourceCLI {
		t.Fatalf("max cycles = %d (%s)", cfg.MaxCycles, source(cfg, "max-cycles"))
	}
	cfg, err = Load(root, home, Overrides{})
	if err != nil || cfg.MaxCycles != 4 || source(cfg, "max-cycles") != SourceProject {
		t.Fatalf("project precedence = %d (%s), %v", cfg.MaxCycles, source(cfg, "max-cycles"), err)
	}
	if !strings.Contains(Format(cfg), "max-cycles: 4 (project; built-in: 10)") {
		t.Fatalf("formatted config:\n%s", Format(cfg))
	}
}

func TestPromptResolutionAndMissingExplicitPath(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	write(t, filepath.Join(home, ".reviewer", "fix-findings.md"), "user prompt\n")
	cfg, err := Load(root, home, Overrides{})
	if err != nil || cfg.FixPrompt != "user prompt\n" || source(cfg, "fix-prompt") != SourceUser {
		t.Fatalf("prompt = %q (%s), %v", cfg.FixPrompt, source(cfg, "fix-prompt"), err)
	}
	_, err = Load(root, home, Overrides{FixPromptPath: OptionalString{Value: "missing.md", Set: true}})
	if err == nil || !strings.Contains(err.Error(), "missing.md") {
		t.Fatalf("missing path error = %v", err)
	}
}

func TestValidationAndGitRoot(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(child, home, Overrides{MaxCycles: OptionalString{Value: "-1", Set: true}})
	if err == nil || cfg.Root != "" {
		t.Fatalf("negative value accepted: %#v, %v", cfg, err)
	}
	cfg, err = Load(child, home, Overrides{})
	physicalRoot, physicalErr := filepath.EvalSymlinks(root)
	if err != nil || physicalErr != nil || cfg.Root != physicalRoot {
		t.Fatalf("root = %s, want %s: %v / %v", cfg.Root, physicalRoot, err, physicalErr)
	}
	if _, err := FindGitRoot(t.TempDir()); err == nil {
		t.Fatal("non-git directory accepted")
	}
	fake := t.TempDir()
	if err := os.Mkdir(filepath.Join(fake, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := FindGitRoot(fake); err == nil {
		t.Fatal("empty .git marker accepted")
	}
}

func source(cfg Config, name string) string {
	for _, setting := range cfg.Settings {
		if setting.Name == name {
			return setting.Source
		}
	}
	return ""
}

func TestLoadEmptyModelValidation(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	t.Setenv("REVIEWER_REVIEW_MODEL", "   ")
	_, err := Load(root, home, Overrides{})
	if err == nil || !strings.Contains(err.Error(), "review-model must not be empty") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveFileReadError(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	path := filepath.Join(home, ".reviewer", "max-cycles")
	write(t, path, "5\n")
	if err := os.Chmod(path, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0o600)
	_, err := Load(root, home, Overrides{})
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestReadExplicitPromptAbsolutePath(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	tmp := t.TempDir()
	path := filepath.Join(tmp, "prompt.md")
	if err := os.WriteFile(path, []byte("absolute prompt\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(root, home, Overrides{FixPromptPath: OptionalString{Value: path, Set: true}})
	if err != nil || cfg.FixPrompt != "absolute prompt\n" {
		t.Fatalf("prompt = %q, %v", cfg.FixPrompt, err)
	}
}

func TestFormatDefaultSource(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	cfg, err := Load(root, home, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	formatted := Format(cfg)
	for _, line := range strings.Split(formatted, "\n") {
		if line != "" && strings.Contains(line, "built-in:") {
			t.Fatalf("default source should not show built-in: %q", line)
		}
	}
}
