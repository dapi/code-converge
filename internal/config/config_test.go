package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

var reviewerEnv = []string{
	"REVIEWER_MODE",
	"REVIEWER_MAX_CYCLES", "REVIEWER_MAX_CI_RECOVERIES", "REVIEWER_REVIEW_MODEL", "REVIEWER_REVIEW_REASONING_EFFORT",
	"REVIEWER_FIX_MODEL", "REVIEWER_FIX_REASONING_EFFORT", "REVIEWER_FIX_PROMPT_FILE", "REVIEWER_FINALIZE_MODEL",
	"REVIEWER_FINALIZE_REASONING_EFFORT", "REVIEWER_FINALIZE_PROMPT_FILE", "REVIEWER_CI_FIX_MODEL",
	"REVIEWER_CI_FIX_REASONING_EFFORT", "REVIEWER_CI_FIX_PROMPT_FILE",
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

func TestProfileResolution(t *testing.T) {
	tests := []struct {
		name      string
		overrides Overrides
		wantMode  string
		want      []string
	}{
		{
			name: "default fast", wantMode: "fast",
			want: []string{"gpt-5.6-terra", "medium", "gpt-5.6-luna", "medium", "gpt-5.6-luna", "medium", "gpt-5.6-luna", "medium"},
		},
		{
			name: "explicit best", overrides: Overrides{Mode: OptionalString{Value: "best", Set: true}}, wantMode: "best",
			want: []string{"gpt-5.6-sol", "high", "gpt-5.6-terra", "high", "gpt-5.6-luna", "medium", "gpt-5.6-terra", "high"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanEnv(t)
			root, home := repo(t)
			cfg, err := Load(root, home, test.overrides)
			if err != nil {
				t.Fatal(err)
			}
			got := []string{cfg.ReviewModel, cfg.ReviewEffort, cfg.FixModel, cfg.FixEffort, cfg.FinalizeModel, cfg.FinalizeEffort, cfg.CIFixModel, cfg.CIFixEffort}
			if cfg.Mode != test.wantMode || strings.Join(got, "|") != strings.Join(test.want, "|") {
				t.Fatalf("mode/profile = %s %q, want %s %q", cfg.Mode, got, test.wantMode, test.want)
			}
			for _, name := range []string{"review-model", "review-reasoning-effort", "fix-model", "fix-reasoning-effort", "finalize-model", "finalize-reasoning-effort", "ci-fix-model", "ci-fix-reasoning-effort"} {
				if gotSource := source(cfg, name); gotSource != test.wantMode+" profile" {
					t.Errorf("%s source = %q", name, gotSource)
				}
			}
		})
	}
}

func TestModePrecedence(t *testing.T) {
	tests := []struct {
		name     string
		want     string
		wantMode string
		set      func(*testing.T, string, string, *Overrides)
	}{
		{"environment", SourceEnv, "best", func(t *testing.T, _, _ string, _ *Overrides) { t.Setenv("REVIEWER_MODE", "best") }},
		{"user", SourceUser, "fast", func(t *testing.T, _, home string, _ *Overrides) {
			t.Setenv("REVIEWER_MODE", "best")
			write(t, filepath.Join(home, ".reviewer", "mode"), "fast")
		}},
		{"project", SourceProject, "fast", func(t *testing.T, root, home string, _ *Overrides) {
			t.Setenv("REVIEWER_MODE", "fast")
			write(t, filepath.Join(home, ".reviewer", "mode"), "best")
			write(t, filepath.Join(root, ".reviewer", "mode"), "fast")
		}},
		{"cli", SourceCLI, "fast", func(t *testing.T, root, home string, overrides *Overrides) {
			t.Setenv("REVIEWER_MODE", "best")
			write(t, filepath.Join(home, ".reviewer", "mode"), "fast")
			write(t, filepath.Join(root, ".reviewer", "mode"), "best")
			overrides.Mode = OptionalString{Value: "fast", Set: true}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cleanEnv(t)
			root, home := repo(t)
			overrides := Overrides{}
			test.set(t, root, home, &overrides)
			cfg, err := Load(root, home, overrides)
			if err != nil || cfg.Mode != test.wantMode || source(cfg, "mode") != test.want {
				t.Fatalf("mode = %q (%s), %v", cfg.Mode, source(cfg, "mode"), err)
			}
		})
	}
}

func TestEveryStageOverrideSourceBeatsProfile(t *testing.T) {
	type field struct {
		name string
		env  string
		set  func(*Overrides, string)
		get  func(Config) string
	}
	fields := []field{
		{"review-model", "REVIEWER_REVIEW_MODEL", func(o *Overrides, v string) { o.ReviewModel = OptionalString{v, true} }, func(c Config) string { return c.ReviewModel }},
		{"review-reasoning-effort", "REVIEWER_REVIEW_REASONING_EFFORT", func(o *Overrides, v string) { o.ReviewEffort = OptionalString{v, true} }, func(c Config) string { return c.ReviewEffort }},
		{"fix-model", "REVIEWER_FIX_MODEL", func(o *Overrides, v string) { o.FixModel = OptionalString{v, true} }, func(c Config) string { return c.FixModel }},
		{"fix-reasoning-effort", "REVIEWER_FIX_REASONING_EFFORT", func(o *Overrides, v string) { o.FixEffort = OptionalString{v, true} }, func(c Config) string { return c.FixEffort }},
		{"finalize-model", "REVIEWER_FINALIZE_MODEL", func(o *Overrides, v string) { o.FinalizeModel = OptionalString{v, true} }, func(c Config) string { return c.FinalizeModel }},
		{"finalize-reasoning-effort", "REVIEWER_FINALIZE_REASONING_EFFORT", func(o *Overrides, v string) { o.FinalizeEffort = OptionalString{v, true} }, func(c Config) string { return c.FinalizeEffort }},
		{"ci-fix-model", "REVIEWER_CI_FIX_MODEL", func(o *Overrides, v string) { o.CIFixModel = OptionalString{v, true} }, func(c Config) string { return c.CIFixModel }},
		{"ci-fix-reasoning-effort", "REVIEWER_CI_FIX_REASONING_EFFORT", func(o *Overrides, v string) { o.CIFixEffort = OptionalString{v, true} }, func(c Config) string { return c.CIFixEffort }},
	}
	sources := []struct {
		name string
		want string
		set  func(*testing.T, string, string, field, *Overrides)
	}{
		{"environment", SourceEnv, func(t *testing.T, _, _ string, f field, _ *Overrides) { t.Setenv(f.env, "environment-value") }},
		{"user", SourceUser, func(t *testing.T, _, home string, f field, _ *Overrides) {
			t.Setenv(f.env, "environment-value")
			write(t, filepath.Join(home, ".reviewer", f.name), "user-value")
		}},
		{"project", SourceProject, func(t *testing.T, root, home string, f field, _ *Overrides) {
			t.Setenv(f.env, "environment-value")
			write(t, filepath.Join(home, ".reviewer", f.name), "user-value")
			write(t, filepath.Join(root, ".reviewer", f.name), "project-value")
		}},
		{"cli", SourceCLI, func(t *testing.T, root, home string, f field, o *Overrides) {
			t.Setenv(f.env, "environment-value")
			write(t, filepath.Join(home, ".reviewer", f.name), "user-value")
			write(t, filepath.Join(root, ".reviewer", f.name), "project-value")
			f.set(o, "cli-value")
		}},
	}
	for _, f := range fields {
		for _, candidate := range sources {
			t.Run(f.name+"/"+candidate.name, func(t *testing.T) {
				cleanEnv(t)
				root, home := repo(t)
				overrides := Overrides{Mode: OptionalString{Value: "best", Set: true}}
				candidate.set(t, root, home, f, &overrides)
				cfg, err := Load(root, home, overrides)
				if err != nil {
					t.Fatal(err)
				}
				if f.get(cfg) != candidate.name+"-value" || source(cfg, f.name) != candidate.want {
					t.Fatalf("%s = %q (%s)", f.name, f.get(cfg), source(cfg, f.name))
				}
			})
		}
	}
}

func TestInvalidModes(t *testing.T) {
	for _, value := range []string{"", "   ", "FAST", "unknown"} {
		t.Run(strconv.Quote(value), func(t *testing.T) {
			cleanEnv(t)
			root, home := repo(t)
			_, err := Load(root, home, Overrides{Mode: OptionalString{Value: value, Set: true}})
			if err == nil || !strings.Contains(err.Error(), "mode must be one of") {
				t.Fatalf("error = %v", err)
			}
		})
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

func TestLoadEmptyStageSettingValidation(t *testing.T) {
	for _, name := range []string{"review-model", "review-reasoning-effort", "fix-model", "fix-reasoning-effort", "finalize-model", "finalize-reasoning-effort", "ci-fix-model", "ci-fix-reasoning-effort"} {
		t.Run(name, func(t *testing.T) {
			cleanEnv(t)
			root, home := repo(t)
			write(t, filepath.Join(root, ".reviewer", name), "   ")
			_, err := Load(root, home, Overrides{})
			if err == nil || !strings.Contains(err.Error(), name+" must not be empty") {
				t.Fatalf("error = %v", err)
			}
		})
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

func TestFormatProfileAndEqualExplicitSources(t *testing.T) {
	cleanEnv(t)
	root, home := repo(t)
	cfg, err := Load(root, home, Overrides{
		Mode:        OptionalString{Value: "best", Set: true},
		ReviewModel: OptionalString{Value: "gpt-5.6-terra", Set: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	formatted := Format(cfg)
	for _, want := range []string{
		"mode: best (cli; built-in: fast)",
		"review-model: gpt-5.6-terra (cli)",
		"fix-model: gpt-5.6-terra (best profile; built-in: gpt-5.6-luna)",
		"finalize-model: gpt-5.6-luna (best profile)",
	} {
		if !strings.Contains(formatted, want) {
			t.Errorf("missing %q in:\n%s", want, formatted)
		}
	}
}
