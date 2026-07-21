package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var markdownLink = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: check-markdown-links FILE...")
		return 2
	}

	failed := false
	for _, source := range args {
		contents, err := os.ReadFile(source)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", source, err)
			failed = true
			continue
		}
		for _, match := range markdownLink.FindAllStringSubmatch(string(contents), -1) {
			target := strings.Trim(strings.TrimSpace(match[1]), "<>")
			if strings.Contains(target, " ") {
				target = strings.Fields(target)[0]
			}
			if target == "" || strings.HasPrefix(target, "#") || hasExternalScheme(target) {
				continue
			}
			target = strings.SplitN(target, "#", 2)[0]
			decoded, err := url.PathUnescape(target)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: invalid link %q: %v\n", source, target, err)
				failed = true
				continue
			}
			resolved := filepath.Clean(filepath.Join(filepath.Dir(source), filepath.FromSlash(decoded)))
			if _, err := os.Stat(resolved); err != nil {
				fmt.Fprintf(os.Stderr, "%s: broken link %q -> %s\n", source, match[1], resolved)
				failed = true
			}
		}
	}

	if failed {
		return 1
	}
	return 0
}

func hasExternalScheme(target string) bool {
	parsed, err := url.Parse(target)
	return err == nil && parsed.Scheme != ""
}
