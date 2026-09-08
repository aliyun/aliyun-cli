package downloadassets

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Execute downloads release assets and prints the committed checksum file.
// A non-zero status means no checksum was published by this run.
func Execute(args []string, stdout, stderr io.Writer) int {
	version, err := parseArgs(args)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	workDir := os.Getenv("DOWNLOAD_ASSETS_WORKDIR")
	if workDir == "" {
		workDir = "."
	}
	opt := Options{
		Version: version,
		BaseURL: os.Getenv("DOWNLOAD_ASSETS_BASE_URL"),
		Token:   os.Getenv("GITHUB_TOKEN"),
		WorkDir: workDir,
	}
	if raw := os.Getenv("DOWNLOAD_ASSETS_TIMEOUT"); raw != "" {
		timeout, err := time.ParseDuration(raw)
		if err != nil {
			fmt.Fprintf(stderr, "download assets: invalid timeout %q\n", raw)
			return 1
		}
		opt.Timeout = timeout
	}
	if err := Download(context.Background(), opt); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	body, err := os.ReadFile(filepath.Join(workDir, checksumName))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if _, err := io.WriteString(stdout, string(body)); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func parseArgs(args []string) (string, error) {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "--" {
			continue
		}
		filtered = append(filtered, arg)
	}
	if len(filtered) != 1 || strings.TrimSpace(filtered[0]) == "" {
		return "", fmt.Errorf("usage: downloadassets <version>")
	}
	return filtered[0], nil
}
