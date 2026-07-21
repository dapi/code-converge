package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type target struct{ os, arch string }

var targets = []target{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}}

func main() {
	version := flag.String("version", "dev", "artifact version")
	outDir := flag.String("out", "dist", "output directory")
	flag.Parse()
	if !validVersion(*version) {
		fatalf("invalid version %q", *version)
	}
	workDir, err := os.Getwd()
	if err != nil {
		fatalf("resolve source directory: %v", err)
	}
	if err := validateOutputDir(*outDir, workDir); err != nil {
		fatalf("invalid output directory: %v", err)
	}
	if err := os.RemoveAll(*outDir); err != nil {
		fatalf("clean output: %v", err)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatalf("create output: %v", err)
	}
	temp, err := os.MkdirTemp("", "reviewer-dist-")
	if err != nil {
		fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(temp)

	checksums := make(map[string]string)
	for _, item := range targets {
		binary := filepath.Join(temp, item.os+"-"+item.arch, "reviewer")
		if err := os.MkdirAll(filepath.Dir(binary), 0o755); err != nil {
			fatalf("create build dir: %v", err)
		}
		ldflags := fmt.Sprintf("-s -w -buildid= -X github.com/dapi/reviewer/internal/version.Version=%s", *version)
		cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags="+ldflags, "-o", binary, "./cmd/reviewer")
		cmd.Env = targetEnv(os.Environ(), item)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			fatalf("build %s/%s: %v", item.os, item.arch, err)
		}
		name := fmt.Sprintf("reviewer_%s_%s_%s.tar.gz", *version, item.os, item.arch)
		archive := filepath.Join(*outDir, name)
		if err := writeArchive(archive, binary); err != nil {
			fatalf("archive %s/%s: %v", item.os, item.arch, err)
		}
		digest, err := hashFile(archive)
		if err != nil {
			fatalf("checksum %s: %v", archive, err)
		}
		checksums[name] = digest
	}

	names := make([]string, 0, len(checksums))
	for name := range checksums {
		names = append(names, name)
	}
	sort.Strings(names)
	var manifest strings.Builder
	for _, name := range names {
		fmt.Fprintf(&manifest, "%s  %s\n", checksums[name], name)
	}
	if err := os.WriteFile(filepath.Join(*outDir, "SHA256SUMS"), []byte(manifest.String()), 0o644); err != nil {
		fatalf("write checksums: %v", err)
	}
}

func validVersion(version string) bool {
	return strings.TrimSpace(version) != "" && !strings.ContainsAny(version, "/\\ \t\r\n")
}

// validateOutputDir prevents the cleanup step from deleting the source checkout.
// Both paths are resolved through their existing symlinks before containment is checked.
func validateOutputDir(outDir, sourceDir string) error {
	output, err := canonicalPath(outDir)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	source, err := canonicalPath(sourceDir)
	if err != nil {
		return fmt.Errorf("resolve source directory: %w", err)
	}
	rel, err := filepath.Rel(output, source)
	if err != nil {
		return fmt.Errorf("compare output and source directories: %w", err)
	}
	if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		return errors.New("output directory must not contain the source checkout")
	}
	return nil
}

// canonicalPath resolves symlinks in the deepest existing parent, preserving any
// not-yet-created path components. This also protects new output directories under
// a symlinked ancestor.
func canonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	for existing := abs; ; existing = filepath.Dir(existing) {
		resolved, err := filepath.EvalSymlinks(existing)
		if err == nil {
			rel, err := filepath.Rel(existing, abs)
			if err != nil {
				return "", err
			}
			return filepath.Join(resolved, rel), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return "", err
		}
	}
}

func targetEnv(base []string, item target) []string {
	env := make([]string, 0, len(base)+4)
	for _, value := range base {
		if strings.HasPrefix(value, "CGO_ENABLED=") || strings.HasPrefix(value, "GOOS=") || strings.HasPrefix(value, "GOARCH=") || strings.HasPrefix(value, "GOAMD64=") {
			continue
		}
		env = append(env, value)
	}
	env = append(env, "CGO_ENABLED=0", "GOOS="+item.os, "GOARCH="+item.arch)
	if item.arch == "amd64" {
		env = append(env, "GOAMD64=v1")
	}
	return env
}

func writeArchive(path, binary string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	gz, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		return err
	}
	gz.Header.ModTime = time.Unix(0, 0).UTC()
	gz.Header.OS = 255
	tw := tar.NewWriter(gz)
	data, err := os.Open(binary)
	if err != nil {
		return err
	}
	defer data.Close()
	info, err := data.Stat()
	if err != nil {
		return err
	}
	header := &tar.Header{Name: "reviewer", Mode: 0o755, Size: info.Size(), ModTime: time.Unix(0, 0).UTC(), Typeflag: tar.TypeReg, Format: tar.FormatUSTAR}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := io.Copy(tw, data); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "reviewer-dist: "+format+"\n", args...)
	os.Exit(1)
}
