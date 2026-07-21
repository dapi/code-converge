package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
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

func TestBuildVersionLdflags(t *testing.T) {
	version := "1.2.3"
	ldflags := "-s -w -buildid= -X github.com/dapi/code-converge/internal/version.Version=" + version
	if !strings.Contains(ldflags, "internal/version.Version=1.2.3") {
		t.Fatalf("ldflags = %q", ldflags)
	}
}

func TestCanonicalPathNonExistentParent(t *testing.T) {
	base := t.TempDir()
	got, err := canonicalPath(filepath.Join(base, "dist", "v1"))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(resolved, "dist", "v1")
	if got != want {
		t.Fatalf("canonicalPath = %q, want %q", got, want)
	}
}

func TestCanonicalPathSymlink(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	link := filepath.Join(base, "link")
	if err := os.Mkdir(real, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got, err := canonicalPath(filepath.Join(link, "dist"))
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(resolved, "dist")
	if got != want {
		t.Fatalf("canonicalPath = %q, want %q", got, want)
	}
}

func TestValidateOutputDirChildAsSource(t *testing.T) {
	source := t.TempDir()
	child := filepath.Join(source, "dist")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateOutputDir(child, source); err != nil {
		t.Fatalf("child output dir should be valid: %v", err)
	}
}

func TestCanonicalPathCyclicSymlink(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a")
	b := filepath.Join(base, "b")
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}
	_, err := canonicalPath(filepath.Join(a, "dist"))
	if err == nil {
		t.Fatal("expected cyclic symlink error")
	}
}

func TestHashFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data")
	if err := os.WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := hashFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824" {
		t.Fatalf("hash = %q", got)
	}
}

func TestHashFileMissing(t *testing.T) {
	_, err := hashFile(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestWriteArchiveMissingBinary(t *testing.T) {
	if err := writeArchive(filepath.Join(t.TempDir(), "out.tar.gz"), filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error for missing binary")
	}
}

func TestFatalfExitsWithOne(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-version=bad/version")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got output:\n%s", out)
	}
	if !strings.Contains(string(out), "code-converge-dist:") {
		t.Fatalf("missing fatalf prefix:\n%s", out)
	}
}

func TestWriteArchive(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "code-converge")
	if err := os.WriteFile(binary, []byte("fake binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	archive := filepath.Join(dir, "out.tar.gz")
	if err := writeArchive(archive, binary); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	header, err := tr.Next()
	if err != nil {
		t.Fatal(err)
	}
	if header.Name != "code-converge" || header.Mode != 0o755 || header.Typeflag != tar.TypeReg {
		t.Fatalf("unexpected header: %+v", header)
	}
	content, err := io.ReadAll(tr)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "fake binary" {
		t.Fatalf("unexpected content: %q", content)
	}
}
