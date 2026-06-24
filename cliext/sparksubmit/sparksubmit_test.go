package sparksubmit

import (
	"bytes"
	"os"
	"path/filepath"
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
	v, err := GetLatestVersion("https://example.com/spark")
	if err != nil {
		// network may fail in CI; only assert fallback when unreachable
		if v != defaultToolVersion {
			t.Fatalf("fallback version = %q, want %q", v, defaultToolVersion)
		}
		return
	}
	if strings.TrimSpace(v) == "" {
		t.Fatal("empty version")
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
