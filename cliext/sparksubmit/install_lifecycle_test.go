package sparksubmit

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/require"
)

func sparkLifecycleContext(t *testing.T) (*Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	for _, name := range []string{EnvExecPath, EnvDownloadBaseURL, EnvNoUpdateCheck, EnvEndpoint, EnvWorkspaceID, "SPARK_CONF_DIR"} {
		t.Setenv(name, "")
	}
	directory := t.TempDir()
	previousPath, previousTime := getConfigurePathFunc, timeNowFunc
	getConfigurePathFunc = func() string { return directory }
	timeNowFunc = func() time.Time { return time.Unix(1_700_000_000, 123) }
	t.Cleanup(func() { getConfigurePathFunc, timeNowFunc = previousPath, previousTime })
	var stdout, stderr bytes.Buffer
	c := NewContext(cli.NewCommandContext(&stdout, &stderr))
	c.initBasicInfo()
	return c, &stdout, &stderr
}

func sparkLifecycleZIP(t *testing.T, root string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range []struct{ name, body string }{
		{root + "/", ""},
		{root + "/bin/spark-submit", "spark submit executable"},
		{root + "/bin/spark-sql", "spark sql executable"},
		{root + "/conf/default.properties", "defaults=true\n"},
	} {
		file, err := writer.Create(entry.name)
		require.NoError(t, err)
		_, err = io.WriteString(file, entry.body)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

type sparkLifecycleBody struct {
	io.Reader
	closed bool
}

func (b *sparkLifecycleBody) Close() error { b.closed = true; return nil }

func TestSparkInstallDownloadsReplacesAndCachesVersions(t *testing.T) {
	c, _, _ := sparkLifecycleContext(t)
	t.Setenv(EnvDownloadBaseURL, "https://download.example.invalid/spark/")
	originalLatest, originalGet := getLatestVersionFunc, httpGetFunc
	t.Cleanup(func() { getLatestVersionFunc, httpGetFunc = originalLatest, originalGet })
	latest, versionChecks, downloads := "1.17.0", 0, 0
	getLatestVersionFunc = func(baseURL string) (string, error) {
		versionChecks++
		require.Equal(t, "https://download.example.invalid/spark", baseURL)
		return latest, nil
	}
	var bodies []*sparkLifecycleBody
	httpGetFunc = func(url string) (*http.Response, error) {
		downloads++
		require.Equal(t, "https://download.example.invalid/spark/emr-serverless-spark-tool-"+latest+"-bin.zip", url)
		body := &sparkLifecycleBody{Reader: bytes.NewReader(sparkLifecycleZIP(t, "emr-serverless-spark-tool-"+latest))}
		bodies = append(bodies, body)
		return &http.Response{StatusCode: http.StatusOK, Body: body}, nil
	}

	require.NoError(t, c.ensureInstalledAndUpdated())
	require.True(t, c.installed)
	require.Equal(t, "1.17.0", c.versionLocal)
	require.Equal(t, "1.17.0", c.readLocalVersion())
	installed, err := os.ReadFile(c.execFilePath)
	require.NoError(t, err)
	require.Equal(t, "spark submit executable", string(installed))
	require.NoError(t, os.WriteFile(filepath.Join(c.installDir, "previous-version-only"), []byte("old"), 0600))

	latest = "1.18.0"
	require.True(t, c.cacheStale())
	require.NoError(t, c.ensureInstalledAndUpdated())
	require.Equal(t, "1.18.0", c.readLocalVersion())
	require.NoFileExists(t, filepath.Join(c.installDir, "previous-version-only"))
	require.NoError(t, c.touchCache())
	require.False(t, c.cacheStale())
	require.NoError(t, c.ensureInstalledAndUpdated())
	require.Equal(t, 2, versionChecks)
	require.Equal(t, 2, downloads)
	for _, body := range bodies {
		require.True(t, body.closed)
	}
	entries, err := os.ReadDir(c.configPath)
	require.NoError(t, err)
	require.Len(t, entries, 1, "download, staging and old installation must be removed")
	require.Equal(t, filepath.Base(c.installDir), entries[0].Name())
}

func TestSparkUpdateChecksRespectOverridesAndRecoverFromFailure(t *testing.T) {
	c, _, stderr := sparkLifecycleContext(t)
	originalLatest := getLatestVersionFunc
	t.Cleanup(func() { getLatestVersionFunc = originalLatest })
	checks := 0
	getLatestVersionFunc = func(string) (string, error) {
		checks++
		return "", errors.New("version service unavailable")
	}
	require.ErrorContains(t, c.ensureInstalledAndUpdated(), "resolve spark-submit tool version: version service unavailable")
	require.False(t, c.installed)
	t.Setenv(EnvExecPath, filepath.Join(t.TempDir(), "external-spark-submit"))
	require.NoError(t, c.ensureInstalledAndUpdated())
	require.Equal(t, 1, checks)
	t.Setenv(EnvExecPath, "")
	require.NoError(t, os.MkdirAll(filepath.Dir(c.execFilePath), 0755))
	require.NoError(t, os.WriteFile(c.execFilePath, []byte("cached"), 0755))
	c.initBasicInfo()
	t.Setenv(EnvNoUpdateCheck, "1")
	require.NoError(t, c.ensureInstalledAndUpdated())
	require.Equal(t, 1, checks)
	t.Setenv(EnvNoUpdateCheck, "")
	require.NoError(t, c.ensureInstalledAndUpdated())
	require.Equal(t, 2, checks)
	require.Contains(t, stderr.String(), "update check failed: version service unavailable")
	require.False(t, c.cacheStale(), "failed update checks should also refresh the check timestamp")

	require.NoError(t, os.WriteFile(c.versionCachePath, []byte("invalid timestamp"), 0600))
	require.True(t, c.cacheStale())
	require.NoError(t, os.WriteFile(c.versionFilePath, []byte("1.18.0\n"), 0600))
	getLatestVersionFunc = func(string) (string, error) { return "1.18.0", nil }
	require.NoError(t, c.ensureInstalledAndUpdated())
	require.Empty(t, c.versionRemote, "matching local version needs no installation")
	require.NoError(t, os.Remove(c.versionCachePath))
	require.NoError(t, os.Mkdir(c.versionCachePath, 0755))
	require.True(t, c.cacheStale(), "an unreadable timestamp must trigger a fresh check")
}

type sparkInterruptedDownload struct{}

func (sparkInterruptedDownload) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestSparkFailedInstallCleansDownloadsAndStaging(t *testing.T) {
	for _, test := range []struct {
		name, want string
		status     int
		body       func(*testing.T) io.Reader
	}{
		{name: "status", status: 403, body: func(*testing.T) io.Reader { return strings.NewReader("denied") }, want: "status 403"},
		{name: "interrupted", status: 200, body: func(*testing.T) io.Reader { return sparkInterruptedDownload{} }, want: "unexpected EOF"},
		{name: "invalid zip", status: 200, body: func(*testing.T) io.Reader { return strings.NewReader("invalid zip") }, want: "unzip"},
		{name: "wrong package", status: 200, body: func(t *testing.T) io.Reader { return bytes.NewReader(sparkLifecycleZIP(t, "another-package")) }, want: "missing emr-serverless-spark-tool"},
	} {
		t.Run(test.name, func(t *testing.T) {
			c, _, _ := sparkLifecycleContext(t)
			body := &sparkLifecycleBody{Reader: test.body(t)}
			previousGet := httpGetFunc
			httpGetFunc = func(string) (*http.Response, error) { return &http.Response{StatusCode: test.status, Body: body}, nil }
			t.Cleanup(func() { httpGetFunc = previousGet })
			err := c.downloadAndInstall("1.18.0")
			require.ErrorContains(t, err, test.want)
			require.False(t, c.installed)
			require.True(t, body.closed)
			entries, err := os.ReadDir(c.configPath)
			require.NoError(t, err)
			require.Empty(t, entries, "a failed installation must leave no download or staging directory")
		})
	}
}

func TestSparkInstallRejectsUnusableFilesystemLocations(t *testing.T) {
	c, _, _ := sparkLifecycleContext(t)
	parentFile := filepath.Join(c.configPath, "parent-file")
	require.NoError(t, os.WriteFile(parentFile, []byte("file"), 0600))
	c.installDir = filepath.Join(parentFile, "tool")
	require.Error(t, c.downloadAndInstall("1.18.0"))
	require.Error(t, installFromZip(filepath.Join(c.configPath, "missing.zip"), c.installDir))
	_, err := findToolRoot(filepath.Join(c.configPath, "missing"))
	require.ErrorIs(t, err, os.ErrNotExist)
	originalGet := httpGetFunc
	httpGetFunc = func(string) (*http.Response, error) { return nil, errors.New("download connection failed") }
	t.Cleanup(func() { httpGetFunc = originalGet })
	require.ErrorContains(t, httpGetToFile("https://download.example.invalid/tool.zip", filepath.Join(c.configPath, "tool.zip")), "download connection failed")
	body := &sparkLifecycleBody{Reader: strings.NewReader("unused")}
	httpGetFunc = func(string) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: body}, nil }
	require.Error(t, httpGetToFile("https://download.example.invalid/tool.zip", filepath.Join(parentFile, "tool.zip")))
	require.True(t, body.closed)
}
