package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBumpVersion(t *testing.T) {
	for _, test := range []struct {
		kind string
		want string
	}{{"patch", "1.2.4"}, {"minor", "1.3.0"}, {"major", "2.0.0"}} {
		t.Run(test.kind, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Mkdir(filepath.Join(root, "scripts"), 0o755); err != nil {
				t.Fatal(err)
			}
			script, err := os.ReadFile("bump-version")
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "scripts", "bump-version"), script, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("1.2.3\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("bash", filepath.Join(root, "scripts", "bump-version"), test.kind)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("bump-version: %v\n%s", err, out)
			}
			got, err := os.ReadFile(filepath.Join(root, "VERSION"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(string(got)) != test.want || strings.TrimSpace(string(out)) != test.want {
				t.Fatalf("VERSION=%q output=%q, want %q", got, out, test.want)
			}
		})
	}
}

func TestBumpVersionRejectsInvalidInput(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	script, err := os.ReadFile("bump-version")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", "bump-version"), script, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "VERSION"), []byte("dev\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("bash", filepath.Join(root, "scripts", "bump-version"), "patch").Run(); err == nil {
		t.Fatal("invalid VERSION was accepted")
	}
}
