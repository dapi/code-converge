// Package update discovers and safely installs compatible code-converge releases.
package update

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	latestReleaseURL = "https://api.github.com/repos/dapi/code-converge/releases/latest"
	archiveBinary    = "code-converge"
)

// Runner is the app-facing update command boundary.
type Runner interface {
	Run(context.Context, bool) int
}

// Service implements the update command. Its dependencies are fields to keep
// release and filesystem failure paths deterministic in tests.
type Service struct {
	Version    string
	GOOS       string
	GOARCH     string
	Executable func() (string, error)
	Client     *http.Client
	LatestURL  string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
	Rename     func(string, string) error
}

type release struct {
	TagName    string  `json:"tag_name"`
	Body       string  `json:"body"`
	HTMLURL    string  `json:"html_url"`
	Prerelease bool    `json:"prerelease"`
	Assets     []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type semver struct{ major, minor, patch int }

func (v semver) String() string { return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch) }

func (v semver) compare(other semver) int {
	if v.major != other.major {
		if v.major < other.major {
			return -1
		}
		return 1
	}
	if v.minor != other.minor {
		if v.minor < other.minor {
			return -1
		}
		return 1
	}
	if v.patch != other.patch {
		if v.patch < other.patch {
			return -1
		}
		return 1
	}
	return 0
}

// Run returns 0 for an up-to-date or declined update and 2 for every updater
// operational failure. It never changes the executable before final rename.
func (s Service) Run(ctx context.Context, assumeYes bool) int {
	out, errOut := s.Stdout, s.Stderr
	if out == nil {
		out = os.Stdout
	}
	if errOut == nil {
		errOut = os.Stderr
	}
	fail := func(format string, args ...any) int {
		fmt.Fprintf(errOut, "code-converge update: "+format+"\n", args...)
		return 2
	}

	current, err := parseVersion(s.Version)
	if err != nil {
		return fail("running version: %v", err)
	}
	osName, arch, err := target(s.GOOS, s.GOARCH)
	if err != nil {
		return fail("%v", err)
	}
	executable := s.Executable
	if executable == nil {
		executable = os.Executable
	}
	destination, err := executable()
	if err != nil {
		return fail("running executable: %v", err)
	}

	rel, err := s.latest(ctx)
	if err != nil {
		return fail("latest release: %v", err)
	}
	if rel.Prerelease {
		return fail("latest release is a prerelease")
	}
	targetVersion, err := parseVersion(rel.TagName)
	if err != nil {
		return fail("latest release version: %v", err)
	}
	if targetVersion.compare(current) <= 0 {
		fmt.Fprintf(out, "code-converge is already up to date (v%s).\n", current)
		return 0
	}

	archiveName := fmt.Sprintf("code-converge_%s_%s_%s.tar.gz", targetVersion, osName, arch)
	archiveURL, sumsURL, ok := releaseAssets(rel.Assets, archiveName)
	if !ok {
		return fail("release v%s is missing %s or SHA256SUMS", targetVersion, archiveName)
	}
	fmt.Fprintf(out, "Update available: v%s\n", targetVersion)
	if notes := strings.TrimSpace(rel.Body); notes != "" {
		fmt.Fprintf(out, "Release notes:\n%s\n", notes)
	} else {
		if rel.HTMLURL == "" {
			return fail("release v%s has neither notes nor a release URL", targetVersion)
		}
		fmt.Fprintf(out, "Release URL: %s\n", rel.HTMLURL)
	}
	if !assumeYes {
		fmt.Fprint(out, "Install update? [y/N]: ")
		in := s.Stdin
		if in == nil {
			in = os.Stdin
		}
		line, readErr := bufio.NewReader(in).ReadString('\n')
		if readErr != nil && !errors.Is(readErr, io.EOF) {
			return fail("confirmation input: %v", readErr)
		}
		answer := strings.TrimSpace(line)
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(out, "Update cancelled.")
			return 0
		}
	}

	if err := s.install(ctx, archiveURL, sumsURL, archiveName, destination); err != nil {
		return fail("install v%s: %v", targetVersion, err)
	}
	fmt.Fprintf(out, "Updated code-converge to v%s at %s\n", targetVersion, destination)
	return 0
}

func (s Service) latest(ctx context.Context) (release, error) {
	var rel release
	url := s.LatestURL
	if url == "" {
		url = latestReleaseURL
	}
	if err := s.getJSON(ctx, url, &rel); err != nil {
		return rel, err
	}
	return rel, nil
}

func (s Service) getJSON(ctx context.Context, url string, target any) error {
	resp, err := s.get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %s", url, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode metadata: %w", err)
	}
	return nil
}

func (s Service) get(ctx context.Context, url string) (*http.Response, error) {
	client := s.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func (s Service) install(ctx context.Context, archiveURL, sumsURL, archiveName, destination string) error {
	tmp, err := os.MkdirTemp("", "code-converge-update-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	archivePath := filepath.Join(tmp, archiveName)
	sumsPath := filepath.Join(tmp, "SHA256SUMS")
	if err := s.download(ctx, archiveURL, archivePath); err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	if err := s.download(ctx, sumsURL, sumsPath); err != nil {
		return fmt.Errorf("download SHA256SUMS: %w", err)
	}
	checksum, err := checksumFor(sumsPath, archiveName)
	if err != nil {
		return err
	}
	if err := verifyFile(archivePath, checksum); err != nil {
		return err
	}
	staged, err := extractBinary(archivePath, filepath.Dir(destination))
	if err != nil {
		return err
	}
	defer os.Remove(staged)
	rename := s.Rename
	if rename == nil {
		rename = os.Rename
	}
	if err := rename(staged, destination); err != nil {
		return fmt.Errorf("replace executable: %w", err)
	}
	return nil
}

func (s Service) download(ctx context.Context, url, destination string) error {
	resp, err := s.get(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %s", url, resp.Status)
	}
	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func target(osName, arch string) (string, string, error) {
	if osName == "" {
		osName = runtime.GOOS
	}
	if arch == "" {
		arch = runtime.GOARCH
	}
	switch osName + ":" + arch {
	case "darwin:amd64", "linux:amd64":
		return osName, "amd64", nil
	case "darwin:arm64", "linux:arm64":
		return osName, "arm64", nil
	default:
		return "", "", fmt.Errorf("unsupported platform %s/%s", osName, arch)
	}
}

func parseVersion(value string) (semver, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	var v semver
	var extra string
	if _, err := fmt.Sscanf(value, "%d.%d.%d%s", &v.major, &v.minor, &v.patch, &extra); err == nil || !errors.Is(err, io.EOF) || extra != "" || v.major < 0 || v.minor < 0 || v.patch < 0 {
		return semver{}, fmt.Errorf("invalid semantic version %q", value)
	}
	if v.String() != value {
		return semver{}, fmt.Errorf("invalid semantic version %q", value)
	}
	return v, nil
}

func releaseAssets(assets []asset, archiveName string) (string, string, bool) {
	var archiveURL, sumsURL string
	for _, a := range assets {
		if a.Name == archiveName {
			archiveURL = a.BrowserDownloadURL
		}
		if a.Name == "SHA256SUMS" {
			sumsURL = a.BrowserDownloadURL
		}
	}
	return archiveURL, sumsURL, archiveURL != "" && sumsURL != ""
}

func checksumFor(path, archiveName string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var found string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[1] == archiveName {
			if found != "" || len(fields[0]) != sha256.Size*2 {
				return "", fmt.Errorf("invalid checksum entry for %s", archiveName)
			}
			if _, err := hex.DecodeString(fields[0]); err != nil {
				return "", fmt.Errorf("invalid checksum entry for %s", archiveName)
			}
			found = strings.ToLower(fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("missing checksum for %s", archiveName)
	}
	return found, nil
}

func verifyFile(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if actual := hex.EncodeToString(h.Sum(nil)); actual != expected {
		return fmt.Errorf("checksum mismatch for %s", filepath.Base(path))
	}
	return nil
}

func extractBinary(archivePath, destinationDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var staged string
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("read archive: %w", err)
		}
		if hdr.Name != archiveBinary {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return "", fmt.Errorf("archive binary is not a regular file")
		}
		if staged != "" {
			return "", fmt.Errorf("archive has duplicate binary")
		}
		out, err := os.CreateTemp(destinationDir, ".code-converge-update-")
		if err != nil {
			return "", err
		}
		staged = out.Name()
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(staged)
			return "", err
		}
		if err := out.Chmod(0o755); err != nil {
			out.Close()
			os.Remove(staged)
			return "", err
		}
		if err := out.Close(); err != nil {
			os.Remove(staged)
			return "", err
		}
	}
	if staged == "" {
		return "", fmt.Errorf("archive is missing %s", archiveBinary)
	}
	return staged, nil
}
