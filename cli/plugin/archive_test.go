package plugin

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWritePluginArchiveResponseEnforcesStreamingLimit(t *testing.T) {
	t.Run("exact limit succeeds", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "plugin.tgz")
		resp := &http.Response{
			Body:          io.NopCloser(strings.NewReader("12345")),
			ContentLength: -1,
		}
		require.NoError(t, writePluginArchiveResponse(resp, dest, 5))
		assert.FileExists(t, dest)
	})

	t.Run("chunked body over limit is removed", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "plugin.tgz")
		resp := &http.Response{
			Body:          io.NopCloser(strings.NewReader("123456")),
			ContentLength: -1,
		}
		err := writePluginArchiveResponse(resp, dest, 5)
		require.ErrorContains(t, err, "exceeds limit")
		_, statErr := os.Stat(dest)
		assert.True(t, os.IsNotExist(statErr))
	})

	t.Run("content length over limit is rejected", func(t *testing.T) {
		dest := filepath.Join(t.TempDir(), "plugin.tgz")
		resp := &http.Response{
			Body:          io.NopCloser(strings.NewReader("small")),
			ContentLength: 6,
		}
		err := writePluginArchiveResponse(resp, dest, 5)
		require.ErrorContains(t, err, "size 6 exceeds limit")
		_, statErr := os.Stat(dest)
		assert.True(t, os.IsNotExist(statErr))
	})
}

func TestUntarResourceLimits(t *testing.T) {
	tests := []struct {
		name    string
		files   []testFile
		limits  pluginArchiveLimits
		wantErr string
	}{
		{
			name:   "exact limits succeed",
			files:  []testFile{{name: "file", content: "12345"}},
			limits: pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 5, extractedBytes: 5, entries: 1},
		},
		{
			name:    "single file exceeds limit",
			files:   []testFile{{name: "file", content: "123456"}},
			limits:  pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 5, extractedBytes: 10, entries: 2},
			wantErr: "per-file limit",
		},
		{
			name: "total extracted size exceeds limit",
			files: []testFile{
				{name: "first", content: "12345"},
				{name: "second", content: "6789"},
			},
			limits:  pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 5, extractedBytes: 8, entries: 2},
			wantErr: "extracted size exceeds limit",
		},
		{
			name: "entry count exceeds limit",
			files: []testFile{
				{name: "one/", isDir: true},
				{name: "two/", isDir: true},
			},
			limits:  pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 10, extractedBytes: 10, entries: 1},
			wantErr: "more than 1 entries",
		},
		{
			name:    "compressed expansion exceeds total limit",
			files:   []testFile{{name: "zeros", content: strings.Repeat("0", 4096)}},
			limits:  pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 8192, extractedBytes: 1024, entries: 1},
			wantErr: "extracted size exceeds limit",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "plugin.tgz")
			require.NoError(t, createTestTarGz(archivePath, test.files))
			err := untarWithLimits(archivePath, filepath.Join(t.TempDir(), "extract"), test.limits)
			if test.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantErr)
			}
		})
	}
}

func TestUnzipResourceLimits(t *testing.T) {
	tests := []struct {
		name    string
		files   []testFile
		limits  pluginArchiveLimits
		wantErr string
	}{
		{
			name:   "exact limits succeed",
			files:  []testFile{{name: "file", content: "12345"}},
			limits: pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 5, extractedBytes: 5, entries: 1},
		},
		{
			name:    "single file exceeds limit",
			files:   []testFile{{name: "file", content: "123456"}},
			limits:  pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 5, extractedBytes: 10, entries: 2},
			wantErr: "per-file limit",
		},
		{
			name: "total extracted size exceeds limit",
			files: []testFile{
				{name: "first", content: "12345"},
				{name: "second", content: "6789"},
			},
			limits:  pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 5, extractedBytes: 8, entries: 2},
			wantErr: "extracted size exceeds limit",
		},
		{
			name: "entry count exceeds limit",
			files: []testFile{
				{name: "one/", isDir: true},
				{name: "two/", isDir: true},
			},
			limits:  pluginArchiveLimits{archiveBytes: 1 << 20, fileBytes: 10, extractedBytes: 10, entries: 1},
			wantErr: "more than 1 entries",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archivePath := filepath.Join(t.TempDir(), "plugin.zip")
			require.NoError(t, createTestZip(archivePath, test.files))
			err := unzipWithLimits(archivePath, filepath.Join(t.TempDir(), "extract"), test.limits)
			if test.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.wantErr)
			}
		})
	}
}

func TestExtractPluginFailureCleansStaging(t *testing.T) {
	root := t.TempDir()
	archivePath := filepath.Join(root, "broken.tgz")
	require.NoError(t, os.WriteFile(archivePath, []byte("not gzip"), 0644))
	extractDir := filepath.Join(root, "staging")
	require.NoError(t, os.MkdirAll(extractDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(extractDir, "partial"), []byte("partial"), 0644))

	err := (&Manager{rootDir: root}).extractPlugin(archivePath, extractDir, archivePath)
	require.Error(t, err)
	_, statErr := os.Stat(extractDir)
	assert.True(t, os.IsNotExist(statErr))
}

func TestOversizedLocalPackagePreservesExistingPlugin(t *testing.T) {
	root := t.TempDir()
	pluginDir := filepath.Join(root, "limited-plugin")
	require.NoError(t, os.MkdirAll(pluginDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(pluginDir, "old-binary"), []byte("old"), 0755))

	archive := createTestPluginArchive(t, "limited-plugin", "2.0.0", "limited")
	archivePath := filepath.Join(t.TempDir(), "limited-plugin.tgz")
	require.NoError(t, os.WriteFile(archivePath, archive, 0644))
	limits := defaultPluginArchiveLimits
	limits.archiveBytes = int64(len(archive) - 1)
	mgr := &Manager{rootDir: root, archiveLimits: &limits}
	require.NoError(t, mgr.saveLocalManifest(&LocalManifest{Plugins: map[string]LocalPlugin{
		"limited-plugin": {
			Name: "limited-plugin", Version: "1.0.0", Path: pluginDir, Command: "limited",
		},
	}}))

	err := mgr.installFromPackageFile(newTestContext(), archivePath, archivePath)
	require.ErrorContains(t, err, "plugin archive size")
	assert.FileExists(t, filepath.Join(pluginDir, "old-binary"))
	manifest, manifestErr := mgr.GetLocalManifest()
	require.NoError(t, manifestErr)
	assert.Equal(t, "1.0.0", manifest.Plugins["limited-plugin"].Version)
}
