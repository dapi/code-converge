package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHasExternalScheme(t *testing.T) {
	for _, target := range []string{"https://example.com", "http://x", "mailto:a@b"} {
		if !hasExternalScheme(target) {
			t.Errorf("expected external: %q", target)
		}
	}
	for _, target := range []string{"./file.md", "../other.md", "file.md#section", ""} {
		if hasExternalScheme(target) {
			t.Errorf("expected internal: %q", target)
		}
	}
}

func makeFixture(t *testing.T) (source, existing string) {
	t.Helper()
	dir := t.TempDir()
	existing = filepath.Join(dir, "existing.md")
	if err := os.WriteFile(existing, []byte("# Existing\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	source = filepath.Join(dir, "source.md")
	return source, existing
}

func TestCheckMarkdownLinks(t *testing.T) {
	source, _ := makeFixture(t)
	content := "[valid](existing.md) [broken](missing.md) [external](https://example.com) [anchor](#section) [percent](existing%2Emd)"
	if err := os.WriteFile(source, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{source}); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
}

func TestCheckMarkdownLinksValid(t *testing.T) {
	source, _ := makeFixture(t)
	content := "[valid](existing.md) [external](https://example.com) [anchor](#section)"
	if err := os.WriteFile(source, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{source}); code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
}

func TestCheckMarkdownLinksNoArgs(t *testing.T) {
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got output:\n%s", out)
	}
	if !strings.Contains(string(out), "usage") {
		t.Fatalf("missing usage message:\n%s", out)
	}
}
