package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
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
		cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid=", "-o", binary, "./cmd/reviewer")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+item.os, "GOARCH="+item.arch)
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
