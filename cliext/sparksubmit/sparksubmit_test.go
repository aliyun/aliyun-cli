package sparksubmit

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
)

func TestNewSparkSubmitCommand(t *testing.T) {
	cmd := NewSparkSubmitCommand()
	if cmd == nil {
		t.Fatal("nil command")
	}
	if cmd.Name != "spark-submit" {
		t.Fatalf("name = %q, want spark-submit", cmd.Name)
	}
	if !cmd.EnableUnknownFlag || !cmd.KeepArgs || !cmd.SkipDefaultHelp {
		t.Fatalf("unexpected command flags: %+v", cmd)
	}
}

func TestPackageZipURL(t *testing.T) {
	c := NewContext(cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{}))
	t.Setenv(EnvDownloadBaseURL, "https://example.com/spark")
	got := c.packageZipURL("1.16.0")
	want := "https://example.com/spark/emr-serverless-spark-tool-1.16.0-bin.zip"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDefaultDownloadURLs(t *testing.T) {
	c := NewContext(cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{}))
	t.Setenv(EnvDownloadBaseURL, "")

	zipURL := c.packageZipURL(defaultToolVersion)
	wantZip := "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/cli-ext/spark-submit/emr-serverless-spark-tool-1.16.0-bin.zip"
	if zipURL != wantZip {
		t.Fatalf("zip URL = %q, want %q", zipURL, wantZip)
	}

	wantVersionURL := "https://aliyun-cli-pub.oss-cn-hangzhou.aliyuncs.com/cli-ext/spark-submit/version.txt"
	gotVersionURL := strings.TrimRight(c.effectiveBaseURL(), "/") + "/version.txt"
	if gotVersionURL != wantVersionURL {
		t.Fatalf("version URL = %q, want %q", gotVersionURL, wantVersionURL)
	}
}

func TestGetLatestVersionParsesBody(t *testing.T) {
	for _, tt := range []struct {
		name       string
		status     int
		body, want string
	}{
		{"version", 200, "  1.17.2\n", "1.17.2"},
		{"empty", 200, "", defaultToolVersion},
		{"whitespace", 200, " \n\t", defaultToolVersion},
		{"not found", 404, "not found", defaultToolVersion},
		{"server error", 500, "failure", defaultToolVersion},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/spark/version.txt" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if got := r.Header.Get("User-Agent"); got != "aliyun-cli/"+cli.Version {
					t.Errorf("User-Agent = %q", got)
				}
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer server.Close()
			v, err := GetLatestVersion(server.URL + "/spark/")
			if err != nil || v != tt.want {
				t.Fatalf("version = %q, err = %v; want %q", v, err, tt.want)
			}
		})
	}
}

func TestGetLatestVersionConnectionError(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	v, err := GetLatestVersion(server.URL)
	if err == nil || v != "" {
		t.Fatalf("expected empty version and connection error, got %q, %v", v, err)
	}
}

func TestGetLatestVersionReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = io.WriteString(w, "1.17")
	}))
	defer server.Close()
	v, err := GetLatestVersion(server.URL)
	if !errors.Is(err, io.ErrUnexpectedEOF) || v != "" {
		t.Fatalf("expected empty version and truncated-body error, got %q, %v", v, err)
	}
}

func TestWritePropertiesFileMerge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "connection.properties")
	if err := os.WriteFile(path, []byte("workspaceId=w-old\nresourceQueueId=q1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := writePropertiesFile(path, map[string]string{
		"accessKeyId":     "ak",
		"accessKeySecret": "sk",
		"regionId":        "cn-hangzhou",
		"endpoint":        "emr-serverless-spark.cn-hangzhou.aliyuncs.com",
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	props, err := readPropertiesFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if props["workspaceId"] != "w-old" {
		t.Fatalf("workspaceId = %q", props["workspaceId"])
	}
	if props["resourceQueueId"] != "q1" {
		t.Fatalf("resourceQueueId = %q", props["resourceQueueId"])
	}
	if props["accessKeyId"] != "ak" {
		t.Fatalf("accessKeyId = %q", props["accessKeyId"])
	}
}

func TestFindToolRoot(t *testing.T) {
	staging := t.TempDir()
	root := filepath.Join(staging, "emr-serverless-spark-tool-1.16.0")
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findToolRoot(staging)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("got %q, want %q", got, root)
	}
}

func TestEnsureToolBinExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("executable permission bits are not supported on Windows")
	}
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(binDir, "spark-submit")
	if err := os.WriteFile(script, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ensureToolBinExecutable(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(script)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatalf("spark-submit not executable: %o", info.Mode())
	}
}

func TestExtractCredentialsUnsupportedModes(t *testing.T) {
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	for _, mode := range []config.AuthenticateMode{config.Anonymous, config.BearerToken} {
		t.Run(string(mode), func(t *testing.T) {
			_, _, _, err := extractCredentials(ctx, config.Profile{Mode: mode})
			if err == nil {
				t.Fatalf("mode %s should not be supported", mode)
			}
			if !strings.Contains(err.Error(), "not supported") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExtractCredentialsAK(t *testing.T) {
	ctx := cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{})
	id, secret, token, err := extractCredentials(ctx, config.Profile{
		Mode:            config.AK,
		AccessKeyId:     "ak",
		AccessKeySecret: "sk",
	})
	if err != nil {
		t.Fatal(err)
	}
	if id != "ak" || secret != "sk" || token != "" {
		t.Fatalf("got (%q, %q, %q)", id, secret, token)
	}
}

func TestRequireJava(t *testing.T) {
	oldLook := lookPathFunc
	lookPathFunc = func(string) (string, error) { return "", os.ErrNotExist }
	defer func() { lookPathFunc = oldLook }()

	err := requireJava()
	if err == nil || !strings.Contains(err.Error(), "java not found") {
		t.Fatalf("expected java not found error, got %v", err)
	}
}

func TestRemoveFlagsForMainCli(t *testing.T) {
	c := NewContext(cli.NewCommandContext(&bytes.Buffer{}, &bytes.Buffer{}))
	got := c.removeFlagsForMainCli([]string{"--region", "cn-hangzhou", "--name", "job1", "oss://b/a.jar"})
	want := []string{"--name", "job1", "oss://b/a.jar"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
