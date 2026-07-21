package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidVersion(t *testing.T) {
	for _, value := range []string{"dev", "test", "1.0.0-next", "1.0.0-rc1"} {
		if !validVersion(value) {
			t.Errorf("valid version rejected: %q", value)
		}
	}
	for _, value := range []string{"", " ", "../1", `one\\two`, "has space", "line\nbreak", "tab\tvalue"} {
		if validVersion(value) {
			t.Errorf("invalid version accepted: %q", value)
		}
	}
}

func TestValidateOutputDir(t *testing.T) {
	source := t.TempDir()
	parent := filepath.Dir(source)
	for _, test := range []struct {
		name    string
		outDir  string
		wantErr bool
	}{
		{"source", source, true},
		{"ancestor", parent, true},
		{"child", filepath.Join(source, "dist"), false},
		{"sibling", filepath.Join(parent, "artifacts"), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateOutputDir(test.outDir, source)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateOutputDir(%q) error = %v, wantErr %v", test.outDir, err, test.wantErr)
			}
		})
	}
	link := filepath.Join(t.TempDir(), "checkout")
	if err := os.Symlink(source, link); err != nil {
		t.Fatal(err)
	}
	if err := validateOutputDir(link, source); err == nil {
		t.Fatal("symlink to source checkout accepted")
	}
}

func TestTargetEnvPinsAMD64(t *testing.T) {
	base := []string{"PATH=/bin", "GOAMD64=v4", "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=1"}
	amd64 := strings.Join(targetEnv(base, target{os: "linux", arch: "amd64"}), "\n")
	if !strings.Contains(amd64, "GOAMD64=v1") || strings.Contains(amd64, "GOAMD64=v4") {
		t.Fatalf("amd64 environment = %q", amd64)
	}
	arm64 := strings.Join(targetEnv(base, target{os: "linux", arch: "arm64"}), "\n")
	if strings.Contains(arm64, "GOAMD64=") {
		t.Fatalf("arm64 environment inherits GOAMD64: %q", arm64)
	}
}
