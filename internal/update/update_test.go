package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testArchive = "code-converge_1.1.0_linux_amd64.tar.gz"

func TestCurrentReleaseDoesNotDownloadOrReplace(t *testing.T) {
	archive := archiveFixture(t, []byte("new"))
	server, calls := releaseServer(t, "v1.0.0", "notes", archive, true, http.StatusOK)
	defer server.Close()
	destination := writeDestination(t, []byte("old"))
	var stdout, stderr bytes.Buffer
	code := service(server, destination, &stdout, &stderr).Run(context.Background(), false)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "already up to date") || calls.archive != 0 || calls.sums != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q calls=%+v", code, stdout.String(), stderr.String(), calls)
	}
	assertBytes(t, destination, []byte("old"))
}

func TestConfirmationAndVerifiedReplacement(t *testing.T) {
	archive := archiveFixture(t, []byte("new executable"))
	for _, test := range []struct {
		name      string
		input     io.Reader
		yes       bool
		wantBytes []byte
	}{
		{"default declines", strings.NewReader("\n"), false, []byte("old")},
		{"negative declines", strings.NewReader("no\n"), false, []byte("old")},
		{"y installs", strings.NewReader("y\n"), false, []byte("new executable")},
		{"yes installs", strings.NewReader("yes\n"), false, []byte("new executable")},
		{"flag never reads stdin", errReader{}, true, []byte("new executable")},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, _ := releaseServer(t, "v1.1.0", "release notes", archive, true, http.StatusOK)
			defer server.Close()
			destination := writeDestination(t, []byte("old"))
			var stdout, stderr bytes.Buffer
			s := service(server, destination, &stdout, &stderr)
			s.Stdin = test.input
			if code := s.Run(context.Background(), test.yes); code != 0 || stderr.Len() != 0 {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			assertBytes(t, destination, test.wantBytes)
			if test.yes && strings.Contains(stdout.String(), "Install update?") {
				t.Fatalf("--yes prompted: %q", stdout.String())
			}
		})
	}
}

func TestFailuresPreserveOriginalExecutable(t *testing.T) {
	archive := archiveFixture(t, []byte("new"))
	for _, test := range []struct {
		name   string
		setup  func(*Service)
		tag    string
		checks bool
		status int
	}{
		{"unsupported target", func(s *Service) { s.GOARCH = "386" }, "v1.1.0", true, http.StatusOK},
		{"malformed metadata", func(s *Service) {}, "broken", true, http.StatusOK},
		{"download failure", func(s *Service) {}, "v1.1.0", true, http.StatusBadGateway},
		{"missing checksum", func(s *Service) {}, "v1.1.0", false, http.StatusOK},
		{"checksum mismatch", func(s *Service) {}, "v1.1.0", true, http.StatusOK},
		{"rename failure", func(s *Service) { s.Rename = func(string, string) error { return fmt.Errorf("permission denied") } }, "v1.1.0", true, http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, _ := releaseServer(t, test.tag, "", archive, test.checks, test.status)
			defer server.Close()
			destination := writeDestination(t, []byte("old"))
			var stdout, stderr bytes.Buffer
			s := service(server, destination, &stdout, &stderr)
			test.setup(&s)
			if test.name == "checksum mismatch" {
				// A server with a valid checksum is replaced below with one whose sums body is invalid.
				server.Close()
				server = mismatchServer(t, archive)
				s = service(server, destination, &stdout, &stderr)
			}
			if code := s.Run(context.Background(), true); code != 2 || !strings.Contains(stderr.String(), "code-converge update:") {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			assertBytes(t, destination, []byte("old"))
		})
	}
}

func TestMissingAssetAndReleaseURLFallback(t *testing.T) {
	archive := archiveFixture(t, []byte("new"))
	server, _ := releaseServer(t, "v1.1.0", "", archive, true, http.StatusOK)
	defer server.Close()
	destination := writeDestination(t, []byte("old"))
	var stdout, stderr bytes.Buffer
	s := service(server, destination, &stdout, &stderr)
	s.Stdin = strings.NewReader("no\n")
	if code := s.Run(context.Background(), false); code != 0 || !strings.Contains(stdout.String(), "Release URL:") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	var missing *httptest.Server
	missing = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest" {
			fmt.Fprintf(w, `{"tag_name":"v1.1.0","html_url":"%s","assets":[]}`, missing.URL)
			return
		}
		http.NotFound(w, r)
	}))
	defer missing.Close()
	s = service(missing, destination, &stdout, &stderr)
	if code := s.Run(context.Background(), true); code != 2 {
		t.Fatalf("code=%d", code)
	}
	assertBytes(t, destination, []byte("old"))
}

func TestInstalledArchiveReportsTargetVersion(t *testing.T) {
	archive := archiveFixture(t, []byte("#!/bin/sh\necho 'code-converge v1.1.0'\n"))
	server, _ := releaseServer(t, "v1.1.0", "notes", archive, true, http.StatusOK)
	defer server.Close()
	destination := writeDestination(t, []byte("old"))
	var stdout, stderr bytes.Buffer
	if code := service(server, destination, &stdout, &stderr).Run(context.Background(), true); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	out, err := exec.Command(destination, "--version").Output()
	if err != nil || string(out) != "code-converge v1.1.0\n" {
		t.Fatalf("output=%q err=%v", out, err)
	}
}

type counters struct{ archive, sums int }

func releaseServer(t *testing.T, tag, body string, archive []byte, validChecksum bool, archiveStatus int) (*httptest.Server, *counters) {
	t.Helper()
	calls := &counters{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			fmt.Fprintf(w, `{"tag_name":%q,"body":%q,"html_url":%q,"assets":[{"name":%q,"browser_download_url":%q},{"name":"SHA256SUMS","browser_download_url":%q}]}`,
				tag, body, server.URL+"/release/v"+tag, testArchive, server.URL+"/archive", server.URL+"/sums")
		case "/archive":
			calls.archive++
			w.WriteHeader(archiveStatus)
			if archiveStatus == http.StatusOK {
				_, _ = w.Write(archive)
			}
		case "/sums":
			calls.sums++
			sum := sha256.Sum256(archive)
			if !validChecksum {
				fmt.Fprintf(w, "%x  other.tar.gz\n", sum)
			} else {
				fmt.Fprintf(w, "%x  %s\n", sum, testArchive)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	return server, calls
}

func mismatchServer(t *testing.T, archive []byte) *httptest.Server {
	t.Helper()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			fmt.Fprintf(w, `{"tag_name":"v1.1.0","html_url":%q,"assets":[{"name":%q,"browser_download_url":%q},{"name":"SHA256SUMS","browser_download_url":%q}]}`, server.URL+"/release/v1.1.0", testArchive, server.URL+"/archive", server.URL+"/sums")
		case "/archive":
			_, _ = w.Write(archive)
		case "/sums":
			fmt.Fprintf(w, "%s  %s\n", strings.Repeat("0", 64), testArchive)
		}
	}))
	return server
}

func service(server *httptest.Server, destination string, stdout, stderr io.Writer) Service {
	return Service{Version: "1.0.0", GOOS: "linux", GOARCH: "amd64", Client: server.Client(), LatestURL: server.URL + "/latest", Executable: func() (string, error) { return destination, nil }, Stdout: stdout, Stderr: stderr}
}

func archiveFixture(t *testing.T, binary []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: archiveBinary, Mode: 0o755, Size: int64(len(binary)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeDestination(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "code-converge")
	if err := os.WriteFile(path, content, 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("contents=%q want %q", got, want)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("stdin read") }

func TestParseVersion(t *testing.T) {
	for _, value := range []string{"1.2.3", "v1.2.3"} {
		if _, err := parseVersion(value); err != nil {
			t.Errorf("%s: %v", value, err)
		}
	}
	for _, value := range []string{"dev", "1.2", "1.2.3-beta", "01.2.3"} {
		if _, err := parseVersion(value); err == nil {
			t.Errorf("%s accepted", value)
		}
	}
}

func TestChecksumForExactAsset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "SHA256SUMS")
	if err := os.WriteFile(path, []byte(hex.EncodeToString(make([]byte, 32))+"  target\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := checksumFor(path, "target"); err != nil {
		t.Fatal(err)
	}
}
